package dto

import (
	"encoding/xml"
	"io"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type ListObjectsV2Output struct {
	Contents              []Object
	NextContinuationToken *string
}

type Object struct {
	ETag *string

	Key *string

	LastModified *time.Time

	Owner *types.Owner

	Size *int64

	UserMetadata StringMap

	UserTags URLMap `json:"-" xml:"UserTags"`
}

type StringMap map[string]string

func (m *StringMap) UnmarshalXML(d *xml.Decoder, _ xml.StartElement) error {
	*m = StringMap{}
	for {
		// <key>value</key>
		var e struct {
			XMLName xml.Name
			Value   string `xml:",chardata"`
		}
		err := d.Decode(&e)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		tmp := e.Value

		// if tag's value contains a special character like í we need to decode it from mime header
		// https://stackoverflow.com/questions/10828807/how-to-decode-utf-8b-to-string-in-c-sharp
		// https://stackoverflow.com/questions/28932140/converting-utf-8-rfc-2047-to-a-regular-string-in-golang
		if strings.HasPrefix(tmp, "=?UTF-8?b?") {
			dec := new(mime.WordDecoder)
			header, err := dec.DecodeHeader(tmp)
			if err != nil {
				return err
			}
			tmp = header
		}

		(*m)[e.XMLName.Local] = tmp
	}
	return nil
}

type URLMap map[string]string

func (m *URLMap) UnmarshalXML(d *xml.Decoder, se xml.StartElement) error {
	*m = URLMap{}
	var tgs string
	if err := d.DecodeElement(&tgs, &se); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	for tgs != "" {
		var key string
		key, tgs, _ = stringsCut(tgs, "&")
		if key == "" {
			continue
		}
		key, value, _ := stringsCut(key, "=")
		key, err := url.QueryUnescape(key)
		if err != nil {
			return err
		}

		value, err = url.QueryUnescape(value)
		if err != nil {
			return err
		}
		(*m)[key] = value
	}
	return nil
}

func stringsCut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
