// Package mpd implements parsing and generating of MPEG-DASH Media Presentation Description (MPD) files.
package mpd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

// http://mpeg.chiariglione.org/standards/mpeg-dash
// https://www.brendanlong.com/the-structure-of-an-mpeg-dash-mpd.html
// http://standards.iso.org/ittf/PubliclyAvailableStandards/MPEG-DASH_schema_files/DASH-MPD.xsd

var emptyElementRE = regexp.MustCompile(`></[A-Za-z]+>`)

// ConditionalUint (ConditionalUintType) defined in XSD as a union of unsignedInt and boolean.
type ConditionalUint struct {
	U *uint64
	B *bool
}

// MarshalXMLAttr encodes ConditionalUint.
func (c ConditionalUint) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if c.U != nil {
		return xml.Attr{Name: name, Value: strconv.FormatUint(*c.U, 10)}, nil
	}

	if c.B != nil {
		return xml.Attr{Name: name, Value: strconv.FormatBool(*c.B)}, nil
	}

	// both are nil - no attribute, client will threat it like "false"
	return xml.Attr{}, nil
}

// UnmarshalXMLAttr decodes ConditionalUint.
func (c *ConditionalUint) UnmarshalXMLAttr(attr xml.Attr) error {
	u, err := strconv.ParseUint(attr.Value, 10, 64)
	if err == nil {
		c.U = &u
		return nil
	}

	b, err := strconv.ParseBool(attr.Value)
	if err == nil {
		c.B = &b
		return nil
	}

	return fmt.Errorf("ConditionalUint: can't UnmarshalXMLAttr %#v", attr)
}

// check interfaces
var (
	_ xml.MarshalerAttr   = ConditionalUint{}
	_ xml.UnmarshalerAttr = &ConditionalUint{}
)

// MPD represents root XML element.
type MPD struct {
	XMLNS                      *string             `xml:"xmlns,attr"`
	ID                         *string             `xml:"id,attr"`
	Type                       *string             `xml:"type,attr"`
	MinimumUpdatePeriod        *string             `xml:"minimumUpdatePeriod,attr"`
	AvailabilityStartTime      *string             `xml:"availabilityStartTime,attr"`
	MediaPresentationDuration  *string             `xml:"mediaPresentationDuration,attr"`
	MinBufferTime              *string             `xml:"minBufferTime,attr"`
	SuggestedPresentationDelay *string             `xml:"suggestedPresentationDelay,attr"`
	TimeShiftBufferDepth       *string             `xml:"timeShiftBufferDepth,attr"`
	PublishTime                *string             `xml:"publishTime,attr"`
	Profiles                   string              `xml:"profiles,attr"`
	MaxSegmentDuration         *string             `xml:"maxSegmentDuration,attr"`
	Period                     []Period            `xml:"Period,omitempty"`
	BaseURL                    *string             `xml:"BaseURL,omitempty"`
	ProgramInformation         *ProgramInformation `xml:"ProgramInformation,omitempty"`
	SupplementalProperties     []Descriptor        `xml:"SupplementalProperty,omitempty"`
}

// Values for MPD Type attribute
const (
	Static  = "static"
	Dynamic = "dynamic"
)

// Values for ContentComponent contentType attribute
const (
	Video = "video"
	Audio = "audio"
)

// Do not try to use encoding.TextMarshaler and encoding.TextUnmarshaler:
// https://github.com/golang/go/issues/6859#issuecomment-118890463

// Encode generates MPD XML.
func (m *MPD) Encode() ([]byte, error) {
	x := new(bytes.Buffer)
	e := xml.NewEncoder(x)
	e.Indent("", "  ")
	err := e.Encode(m)
	if err != nil {
		return nil, err
	}

	// hacks for self-closing tags
	res := new(bytes.Buffer)
	res.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	res.WriteByte('\n')
	for {
		s, err := x.ReadString('\n')
		if s != "" {
			s = emptyElementRE.ReplaceAllString(s, `/>`)
			res.WriteString(s)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	res.WriteByte('\n')
	return res.Bytes(), err
}

// Decode parses MPD XML.
func (m *MPD) Decode(b []byte) error {
	return xml.Unmarshal(b, m)
}

// ProgramInformation represents XSD's ProgramInformationType
type ProgramInformation struct {
	Lang               *string `xml:"lang,attr"`
	MoreInformationURL *string `xml:"moreInformationURL,attr"`
	Title              *string `xml:"Title,omitempty"`
	Source             *string `xml:"Source,omitempty"`
	Copyright          *string `xml:"Copyright,omitempty"`
}

// Period represents XSD's PeriodType.
type Period struct {
	Start                  *string          `xml:"start,attr"`
	ID                     *string          `xml:"id,attr"`
	Duration               *string          `xml:"duration,attr"`
	AdaptationSets         []*AdaptationSet `xml:"AdaptationSet,omitempty"`
	BaseURL                *string          `xml:"BaseURL,omitempty"`
	AssetIdentifiers       []Descriptor     `xml:"AssetIdentifier,omitempty"`
	SupplementalProperties []Descriptor     `xml:"SupplementalProperty,omitempty"`
}

// ContentComponent represents XSD's ContentComponentType.
type ContentComponent struct {
	ID              *string      `xml:"id,attr"`
	Lang            *string      `xml:"lang,attr"`
	ContentType     *string      `xml:"contentType,attr"`
	Par             *string      `xml:"par,attr"`
	Accessibilities []Descriptor `xml:"Accessibility,omitempty"`
	Roles           []Descriptor `xml:"Role,omitempty"`
	Ratings         []Descriptor `xml:"Rating,omitempty"`
	Viewpoints      []Descriptor `xml:"Viewpoint,omitempty"`
}

// AdaptationSet represents XSD's AdaptationSetType.
type AdaptationSet struct {
	ID                         *string            `xml:"id,attr"`
	MimeType                   string             `xml:"mimeType,attr"`
	SegmentAlignment           ConditionalUint    `xml:"segmentAlignment,attr"`
	SubsegmentAlignment        ConditionalUint    `xml:"subsegmentAlignment,attr"`
	StartWithSAP               *uint64            `xml:"startWithSAP,attr"`
	SubsegmentStartsWithSAP    *uint64            `xml:"subsegmentStartsWithSAP,attr"`
	BitstreamSwitching         *bool              `xml:"bitstreamSwitching,attr"`
	Lang                       *string            `xml:"lang,attr"`
	Width                      *string            `xml:"width,attr"`
	Height                     *string            `xml:"height,attr"`
	MaxWidth                   *uint64            `xml:"maxWidth,attr"`
	MaxHeight                  *uint64            `xml:"maxHeight,attr"`
	MaxFrameRate               *string            `xml:"maxFrameRate,attr"`
	FrameRate                  *string            `xml:"frameRate,attr"`
	Sar                        *string            `xml:"sar,attr"`
	Codecs                     *string            `xml:"codecs,attr"`
	AudioSamplingRate          *string            `xml:"audioSamplingRate,attr"`
	ContentProtections         []Descriptor       `xml:"ContentProtection,omitempty"`
	Representations            []Representation   `xml:"Representation,omitempty"`
	Roles                      []Descriptor       `xml:"Role,omitempty"`
	ContentComponents          []ContentComponent `xml:"ContentComponent,omitempty"`
	BaseURL                    *string            `xml:"BaseURL,omitempty"`
	SegmentTemplate            *SegmentTemplate   `xml:"SegmentTemplate,omitempty"`
	AudioChannelConfigurations []Descriptor       `xml:"AudioChannelConfiguration,omitempty"`
	EssentialProperties        []Descriptor       `xml:"EssentialProperty,omitempty"`
	SupplementalProperties     []Descriptor       `xml:"SupplementalProperty,omitempty"`
}

// SubRepresentation represents the XSD's SubRepresentationType.
type SubRepresentation struct {
	Bandwidth                  *uint64      `xml:"bandwidth,attr"`
	ContentComponent           string       `xml:"ContentComponent,omitempty"`
	AudioSamplingRate          *string      `xml:"audioSamplingRate,attr"`
	Codecs                     *string      `xml:"codecs,attr"`
	AudioChannelConfigurations []Descriptor `xml:"AudioChannelConfiguration,omitempty"`
}

// Representation represents XSD's RepresentationType.
type Representation struct {
	ID                         *string             `xml:"id,attr"`
	Width                      *uint64             `xml:"width,attr"`
	Height                     *uint64             `xml:"height,attr"`
	FrameRate                  *string             `xml:"frameRate,attr"`
	Bandwidth                  *uint64             `xml:"bandwidth,attr"`
	AudioSamplingRate          *string             `xml:"audioSamplingRate,attr"`
	Codecs                     *string             `xml:"codecs,attr"`
	Sar                        *string             `xml:"sar,attr"`
	BaseURL                    *string             `xml:"BaseURL,omitempty"`
	MimeType                   string              `xml:"mimeType,attr"`
	ContentProtections         []Descriptor        `xml:"ContentProtection,omitempty"`
	SegmentTemplate            *SegmentTemplate    `xml:"SegmentTemplate,omitempty"`
	SubRepresentations         []SubRepresentation `xml:"SubRepresentation,omitempty"`
	AudioChannelConfigurations []Descriptor        `xml:"AudioChannelConfiguration,omitempty"`
	EssentialProperties        []Descriptor        `xml:"EssentialProperty,omitempty"`
	SupplementalProperties     []Descriptor        `xml:"SupplementalProperty,omitempty"`
}

// Descriptor represents XSD's DescriptorType.
type Descriptor struct {
	SchemeIDURI string `xml:"schemeIdUri,attr,omitempty"`
	Value       string `xml:"value,attr,omitempty"`
}

// SegmentTemplate represents XSD's SegmentTemplateType.
type SegmentTemplate struct {
	Timescale              *uint64            `xml:"timescale,attr"`
	Media                  *string            `xml:"media,attr"`
	Initialization         *string            `xml:"initialization,attr"`
	StartNumber            *uint64            `xml:"startNumber,attr"`
	PresentationTimeOffset *uint64            `xml:"presentationTimeOffset,attr"`
	SegmentTimelineS       []SegmentTimelineS `xml:"SegmentTimeline>S,omitempty"`
}

// SegmentTimelineS represents XSD's SegmentTimelineType's inner S elements.
type SegmentTimelineS struct {
	T *uint64 `xml:"t,attr"`
	D uint64  `xml:"d,attr"`
	R *uint64 `xml:"r,attr"`
	N *uint64 `xml:"n,attr"`
}
