package index

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Entry struct {
	// either it is direct...
	Title   *string  `json:"title,omitempty"`
	Link    *string  `json:"link,omitempty"`
	Entries *[]Entry `json:"entries,omitempty"`
	// ...or it's indirect
	HRef *string `json:"href,omitempty"`
}

func (e *Entry) String() string {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

type key int8

const (
	count key = iota
)

func Load(ctx context.Context, manifest string) (*Entry, error) {
	// ensure the context has a recursion counter
	if c := ctx.Value(count); c == nil {
		ctx = context.WithValue(ctx, count, 0)
	} else if c, ok := c.(int); ok && c > 10 {
		return nil, errors.New("recursion too deep")
	}

	var (
		data []byte
		err  error
	)
	// read the data
	if manifest == "" {
		return nil, fmt.Errorf("invalid manifest reference: %q", manifest)
	} else if strings.HasPrefix(strings.ToLower(manifest), "https://") || strings.HasPrefix(strings.ToLower(manifest), "http://") {
		data, err = DownloadFile(manifest)
	} else {
		data, err = ioutil.ReadFile(manifest)
	}
	if err != nil {
		return nil, err
	}
	// parse the data
	entry := &Entry{}
	if err := json.Unmarshal(data, entry); err != nil {
		return nil, err
	}
	// validate the data
	if entry.Title != nil && *entry.Title != "" {
		// title is not nil, then this is a direct value:
		// it MAY have a link to the content, and it MAY
		// have some subentries, but it MUSTN'T have a
		// href because it is not a mountpoint!
		if entry.HRef != nil {
			return nil, fmt.Errorf("entry %q must not have a manifest, yet has %q", *entry.Title, *entry.HRef)
		}
		// now recurse on the subentries
		entries := []Entry{}
		for _, subentry := range *entry.Entries {
			if subentry.HRef != nil && *subentry.HRef != "" {
				// this is an indirect entry
				log.Printf("recurse count: %v\n", ctx.Value(count))
				entry, err := Load(context.WithValue(ctx, count, ctx.Value(count).(int)+1), *subentry.HRef)
				if err != nil {
					log.Printf("error unmarshalling href %q", *subentry.HRef)
					continue
				}
				entries = append(entries, *entry)
			} else {
				entries = append(entries, subentry)
			}
		}
		entry.Entries = &entries
	} else if entry.HRef != nil && *entry.HRef != "" {
		log.Println("unmarshalling href")
		// href is not nil, thus we have a mountpoint:
		// we know that title is nil already but so must
		// be the link and the subentries too
		if entry.Link != nil && *entry.Link != "" {
			return nil, fmt.Errorf("mountpoint must not have a link")
		}
		if entry.Entries != nil && len(*entry.Entries) > 0 {
			return nil, fmt.Errorf("mountpoint must not have subentries")
		}
		// if this is a mountpoint, we must
		// load it and recurse
		entry, err = Load(context.WithValue(ctx, count, ctx.Value(count).(int)+1), *entry.HRef)
		if err != nil {
			return nil, err
		}
	}
	return entry, nil
}

func PointerTo[T any](t T) *T {
	return &t
}

func DownloadFile(url string) ([]byte, error) {
	var buffer bytes.Buffer
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if _, err := io.Copy(&buffer, response.Body); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
