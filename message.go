package coap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

const (
	DefaultAckTimeout      = 2
	DefaultAckRandomFactor = 1.5
	DefaultMaxRetransmit   = 4
	DefaultNStart          = 1
	DefaultLeisure         = 5
	DefaultProbingRate     = 1

	DefaultHost  = ""
	DefaultPort  = 5683
	SDefaultPort = 5684

	PayloadMarker = 0xff
	MaxPacketSize = 1500
)

// COAPType represents the message type.
type COAPType uint8

const (
	// Confirmable messages require acknowledgements.
	Confirmable COAPType = 0
	// NonConfirmable messages do not require acknowledgements.
	NonConfirmable COAPType = 1
	// Acknowledgement is a message indicating a response to confirmable message.
	Acknowledgement COAPType = 2
	// Reset indicates a permanent negative acknowledgement.
	Reset COAPType = 3
)

var typeNames = [256]string{
	Confirmable:     "Confirmable",
	NonConfirmable:  "NonConfirmable",
	Acknowledgement: "Acknowledgement",
	Reset:           "Reset",
}

func init() {
	for i := range typeNames {
		if typeNames[i] == "" {
			typeNames[i] = fmt.Sprintf("Unknown (0x%x)", i)
		}
	}
}

func (t COAPType) String() string {
	return typeNames[t]
}

// COAPCode is the type used for both request and response codes.
type COAPCode uint8

const (
	// Request Codes

	GET    COAPCode = 1
	POST   COAPCode = 2
	PUT    COAPCode = 3
	DELETE COAPCode = 4

	// Response Codes

	Empty    COAPCode = 0
	Created  COAPCode = 65 // 2.01
	Deleted  COAPCode = 66 // 2.02
	Valid    COAPCode = 67 // 2.03
	Changed  COAPCode = 68 // 2.04
	Content  COAPCode = 69 // 2.05
	Continue COAPCode = 95 // 2.31

	// 4.x
	BadRequest               COAPCode = 128 // 4.00
	Unauthorized             COAPCode = 129 // 4.01
	BadOption                COAPCode = 130 // 4.02
	Forbidden                COAPCode = 131 // 4.03
	NotFound                 COAPCode = 132 // 4.04
	MethodNotAllowed         COAPCode = 133 // 4.05
	NotAcceptable            COAPCode = 134 // 4.06
	RequestEntityIncomplete  COAPCode = 136 // 4.08
	Conflict                 COAPCode = 137 // 4.09
	PreconditionFailed       COAPCode = 140 // 4.12
	RequestEntityTooLarge    COAPCode = 141 // 4.13
	UnsupportedContentFormat COAPCode = 143 // 4.15

	// 5.x
	InternalServerError  COAPCode = 160 // 5.00
	NotImplemented       COAPCode = 161 // 5.01
	BadGateway           COAPCode = 162 // 5.02
	ServiceUnavailable   COAPCode = 163 // 5.03
	GatewayTimeout       COAPCode = 164 // 5.04
	ProxyingNotSupported COAPCode = 165 // 5.05
)

var codeNames = [256]string{
	GET:                      "GET",
	POST:                     "POST",
	PUT:                      "PUT",
	DELETE:                   "DELETE",
	Created:                  "Created",
	Deleted:                  "Deleted",
	Valid:                    "Valid",
	Changed:                  "Changed",
	Content:                  "Content",
	BadRequest:               "BadRequest",
	Unauthorized:             "Unauthorized",
	BadOption:                "BadOption",
	Forbidden:                "Forbidden",
	NotFound:                 "NotFound",
	MethodNotAllowed:         "MethodNotAllowed",
	NotAcceptable:            "NotAcceptable",
	PreconditionFailed:       "PreconditionFailed",
	RequestEntityTooLarge:    "RequestEntityTooLarge",
	UnsupportedContentFormat: "UnsupportedContentFormat",
	InternalServerError:      "InternalServerError",
	NotImplemented:           "NotImplemented",
	BadGateway:               "BadGateway",
	ServiceUnavailable:       "ServiceUnavailable",
	GatewayTimeout:           "GatewayTimeout",
	ProxyingNotSupported:     "ProxyingNotSupported",
}

func init() {
	for i := range codeNames {
		if codeNames[i] == "" {
			codeNames[i] = fmt.Sprintf("Unknown (0x%x)", i)
		}
	}
}

func (c COAPCode) String() string {
	return codeNames[c]
}

// Message encoding errors.
var (
	ErrInvalidTokenLen     = errors.New("invalid token length")
	ErrOptionTooLong       = errors.New("option is too long")
	ErrOptionGapTooLarge   = errors.New("option gap too large")
	ErrShortPacket         = errors.New("short packet")
	ErrInvalidVersion      = errors.New("invalid version")
	ErrTruncated           = errors.New("truncated")
	ErrInvalidOptionMarker = errors.New("unexpected extended option marker")
)

// OptionID identifies an option in a message.
type OptionID uint8

/*
   +-----+----+---+---+---+----------------+--------+--------+-------------+
   | No. | C  | U | N | R | Name           | Format | Length | Default     |
   +-----+----+---+---+---+----------------+--------+--------+-------------+
   |   1 | x  |   |   | x | If-Match       | opaque | 0-8    | (none)      |
   |   3 | x  | x | - |   | Uri-Host       | string | 1-255  | (see below) |
   |   4 |    |   |   | x | ETag           | opaque | 1-8    | (none)      |
   |   5 | x  |   |   |   | If-None-Match  | empty  | 0      | (none)      |
   |   6 |    | x | - |   | Observe        | uint   | 0-3    | (none)      |
   |   7 | x  | x | - |   | Uri-Port       | uint   | 0-2    | (see below) |
   |   8 |    |   |   | x | Location-Path  | string | 0-255  | (none)      |
   |  11 | x  | x | - | x | Uri-Path       | string | 0-255  | (none)      |
   |  12 |    |   |   |   | Content-Format | uint   | 0-2    | (none)      |
   |  14 |    | x | - |   | Max-Age        | uint   | 0-4    | 60          |
   |  15 | x  | x | - | x | Uri-Query      | string | 0-255  | (none)      |
   |  17 | x  |   |   |   | Accept         | uint   | 0-2    | (none)      |
   |  20 |    |   |   | x | Location-Query | string | 0-255  | (none)      |
   |  23 |    |   | - | - | Block2         | uint   | 0-3    | (none)      |
   |  27 |    |   | - | - | Block1         | uint   | 0-3    | (none)      |
   |  28 |    |   | x |   | Size2          | uint   | 0-4    | (none)      |
   |  35 | x  | x | - |   | Proxy-Uri      | string | 1-1034 | (none)      |
   |  39 | x  | x | - |   | Proxy-Scheme   | string | 1-255  | (none)      |
   |  60 |    |   | x |   | Size1          | uint   | 0-4    | (none)      |
   +-----+----+---+---+---+----------------+--------+--------+-------------+
*/

// Option IDs.
const (
	IfMatch       OptionID = 1
	URIHost       OptionID = 3
	ETag          OptionID = 4
	IfNoneMatch   OptionID = 5
	Observe       OptionID = 6
	URIPort       OptionID = 7
	LocationPath  OptionID = 8
	URIPath       OptionID = 11
	ContentFormat OptionID = 12
	MaxAge        OptionID = 14
	URIQuery      OptionID = 15
	Accept        OptionID = 17
	LocationQuery OptionID = 20
	Block2        OptionID = 23
	Block1        OptionID = 27
	Size2         OptionID = 28
	ProxyURI      OptionID = 35
	ProxyScheme   OptionID = 39
	Size1         OptionID = 60
)

// Option value format (RFC7252 section 3.2)
type valueFormat uint8

const (
	valueUnknown valueFormat = iota
	valueEmpty
	valueOpaque
	valueUint
	valueString
)

type optionDef struct {
	valueFormat valueFormat
	minLen      int
	maxLen      int
}

var optionDefs = [256]optionDef{
	IfMatch:       optionDef{valueFormat: valueOpaque, minLen: 0, maxLen: 8},
	URIHost:       optionDef{valueFormat: valueString, minLen: 1, maxLen: 255},
	ETag:          optionDef{valueFormat: valueOpaque, minLen: 1, maxLen: 8},
	IfNoneMatch:   optionDef{valueFormat: valueEmpty, minLen: 0, maxLen: 0},
	Observe:       optionDef{valueFormat: valueUint, minLen: 0, maxLen: 3},
	URIPort:       optionDef{valueFormat: valueUint, minLen: 0, maxLen: 2},
	LocationPath:  optionDef{valueFormat: valueString, minLen: 0, maxLen: 255},
	URIPath:       optionDef{valueFormat: valueString, minLen: 0, maxLen: 255},
	ContentFormat: optionDef{valueFormat: valueUint, minLen: 0, maxLen: 2},
	MaxAge:        optionDef{valueFormat: valueUint, minLen: 0, maxLen: 4},
	URIQuery:      optionDef{valueFormat: valueString, minLen: 0, maxLen: 255},
	Accept:        optionDef{valueFormat: valueUint, minLen: 0, maxLen: 2},
	LocationQuery: optionDef{valueFormat: valueString, minLen: 0, maxLen: 255},
	Block2:        optionDef{valueFormat: valueUint, minLen: 0, maxLen: 3},
	Block1:        optionDef{valueFormat: valueUint, minLen: 0, maxLen: 3},
	Size2:         optionDef{valueFormat: valueUint, minLen: 0, maxLen: 4},
	ProxyURI:      optionDef{valueFormat: valueString, minLen: 1, maxLen: 1034},
	ProxyScheme:   optionDef{valueFormat: valueString, minLen: 1, maxLen: 255},
	Size1:         optionDef{valueFormat: valueUint, minLen: 0, maxLen: 4},
}

// MediaType specifies the content type of a message.
type MediaType byte

// Content types.
const (
	TextPlain     MediaType = 0  // text/plain;charset=utf-8
	AppLinkFormat MediaType = 40 // application/link-format
	AppXML        MediaType = 41 // application/xml
	AppOctets     MediaType = 42 // application/octet-stream
	AppExi        MediaType = 47 // application/exi
	AppJSON       MediaType = 50 // application/json
)

type option struct {
	ID    OptionID
	Value interface{}
}

func EncodeBlock(NUM, M, SZX uint32) uint32 {
	return NUM<<4 | M<<3 | SZX
}

func DecodeBlock(n uint32) (NUM, M, SZX uint32) {
	return n >> 4, (n >> 3) & 1, n & 7
}

func encodeInt(v uint32) []byte {
	switch {
	case v == 0:
		return nil
	case v < 256:
		return []byte{byte(v)}
	case v < 65536:
		rv := []byte{0, 0}
		binary.BigEndian.PutUint16(rv, uint16(v))
		return rv
	case v < 16777216:
		rv := []byte{0, 0, 0, 0}
		binary.BigEndian.PutUint32(rv, uint32(v))
		return rv[1:]
	default:
		rv := []byte{0, 0, 0, 0}
		binary.BigEndian.PutUint32(rv, uint32(v))
		return rv
	}
}

func decodeInt(b []byte) uint32 {
	tmp := []byte{0, 0, 0, 0}
	copy(tmp[4-len(b):], b)
	return binary.BigEndian.Uint32(tmp)
}

func (o option) toBytes() []byte {
	var v uint32

	switch i := o.Value.(type) {
	case string:
		return []byte(i)
	case []byte:
		return i
	case MediaType:
		v = uint32(i)
	case int:
		v = uint32(i)
	case int32:
		v = uint32(i)
	case uint:
		v = uint32(i)
	case uint32:
		v = i
	default:
		panic(fmt.Errorf("invalid type for option %x: %T (%v)",
			o.ID, o.Value, o.Value))
	}

	return encodeInt(v)
}

func parseOptionValue(optionID OptionID, valueBuf []byte) interface{} {
	def := optionDefs[optionID]
	if def.valueFormat == valueUnknown {
		// Skip unrecognized options (RFC7252 section 5.4.1)
		return nil
	}
	if len(valueBuf) < def.minLen || len(valueBuf) > def.maxLen {
		// Skip options with illegal value length (RFC7252 section 5.4.3)
		return nil
	}
	switch def.valueFormat {
	case valueUint:
		intValue := decodeInt(valueBuf)
		if optionID == ContentFormat || optionID == Accept {
			return MediaType(intValue)
		} else {
			return intValue
		}
	case valueString:
		return string(valueBuf)
	case valueOpaque, valueEmpty:
		return valueBuf
	}
	// Skip unrecognized options (should never be reached)
	return nil
}

type options []option

func (o options) Len() int {
	return len(o)
}

func (o options) Less(i, j int) bool {
	if o[i].ID == o[j].ID {
		return i < j
	}
	return o[i].ID < o[j].ID
}

func (o options) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o options) Minus(oid OptionID) options {
	rv := options{}
	for _, opt := range o {
		if opt.ID != oid {
			rv = append(rv, opt)
		}
	}
	return rv
}

// Message is a CoAP message.
type Message struct {
	Type      COAPType
	Code      COAPCode
	MessageID uint16

	Token, Payload []byte

	opts options
}

// IsConfirmable returns true if this message is confirmable.
func (m Message) IsConfirmable() bool {
	return m.Type == Confirmable
}

// Options gets all the values for the given option.
func (m Message) Options(o OptionID) []interface{} {
	var rv []interface{}

	for _, v := range m.opts {
		if o == v.ID {
			rv = append(rv, v.Value)
		}
	}

	return rv
}

// Option gets the first value for the given option ID.
func (m Message) Option(o OptionID) interface{} {
	for _, v := range m.opts {
		if o == v.ID {
			return v.Value
		}
	}
	return nil
}

func (m Message) optionStrings(o OptionID) []string {
	var rv []string
	for _, o := range m.Options(o) {
		rv = append(rv, o.(string))
	}
	return rv
}

// Path gets the Path set on this message if any.
func (m Message) Path() []string {
	return m.optionStrings(URIPath)
}

// PathString gets a path as a / separated string.
func (m Message) PathString() string {
	return strings.Join(m.Path(), "/")
}

// SetPathString sets a path by a / separated string.
func (m *Message) SetPathString(s string) {
	for s[0] == '/' {
		s = s[1:]
	}
	m.SetPath(strings.Split(s, "/"))
}

// SetPath updates or adds a URIPath attribute on this message.
func (m *Message) SetPath(s []string) {
	m.SetOption(URIPath, s)
}

// RemoveOption removes all references to an option
func (m *Message) RemoveOption(opID OptionID) {
	m.opts = m.opts.Minus(opID)
}

// AddOption adds an option.
func (m *Message) AddOption(opID OptionID, val interface{}) {
	iv := reflect.ValueOf(val)
	if (iv.Kind() == reflect.Slice || iv.Kind() == reflect.Array) &&
		iv.Type().Elem().Kind() == reflect.String {
		for i := 0; i < iv.Len(); i++ {
			m.opts = append(m.opts, option{opID, iv.Index(i).Interface()})
		}
		return
	}
	m.opts = append(m.opts, option{opID, val})
}

// SetOption sets an option, discarding any previous value
func (m *Message) SetOption(opID OptionID, val interface{}) {
	m.RemoveOption(opID)
	m.AddOption(opID, val)
}

const (
	extoptByteCode   = 13
	extoptByteAddend = 13
	extoptWordCode   = 14
	extoptWordAddend = 269
	extoptError      = 15
)

// MarshalBinary produces the binary form of this Message.
func (m *Message) MarshalBinary() ([]byte, error) {
	tmpbuf := []byte{0, 0}
	binary.BigEndian.PutUint16(tmpbuf, m.MessageID)

	/*
	     0                   1                   2                   3
	    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |Ver| T |  TKL  |      Code     |          Message ID           |
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |   Token (if any, TKL bytes) ...
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |   Options (if any) ...
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |1 1 1 1 1 1 1 1|    Payload (if any) ...
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	*/

	buf := bytes.Buffer{}
	buf.Write([]byte{
		(1 << 6) | (uint8(m.Type) << 4) | uint8(0xf&len(m.Token)),
		byte(m.Code),
		tmpbuf[0], tmpbuf[1],
	})
	buf.Write(m.Token)

	/*
	     0   1   2   3   4   5   6   7
	   +---------------+---------------+
	   |               |               |
	   |  Option Delta | Option Length |   1 byte
	   |               |               |
	   +---------------+---------------+
	   \                               \
	   /         Option Delta          /   0-2 bytes
	   \          (extended)           \
	   +-------------------------------+
	   \                               \
	   /         Option Length         /   0-2 bytes
	   \          (extended)           \
	   +-------------------------------+
	   \                               \
	   /                               /
	   \                               \
	   /         Option Value          /   0 or more bytes
	   \                               \
	   /                               /
	   \                               \
	   +-------------------------------+

	   See parseExtOption(), extendOption()
	   and writeOptionHeader() below for implementation details
	*/

	extendOpt := func(opt int) (int, int) {
		ext := 0
		if opt >= extoptByteAddend {
			if opt >= extoptWordAddend {
				ext = opt - extoptWordAddend
				opt = extoptWordCode
			} else {
				ext = opt - extoptByteAddend
				opt = extoptByteCode
			}
		}
		return opt, ext
	}

	writeOptHeader := func(delta, length int) {
		d, dx := extendOpt(delta)
		l, lx := extendOpt(length)

		buf.WriteByte(byte(d<<4) | byte(l))

		tmp := []byte{0, 0}
		writeExt := func(opt, ext int) {
			switch opt {
			case extoptByteCode:
				buf.WriteByte(byte(ext))
			case extoptWordCode:
				binary.BigEndian.PutUint16(tmp, uint16(ext))
				buf.Write(tmp)
			}
		}

		writeExt(d, dx)
		writeExt(l, lx)
	}

	sort.Stable(&m.opts)

	prev := 0

	for _, o := range m.opts {
		b := o.toBytes()
		writeOptHeader(int(o.ID)-prev, len(b))
		buf.Write(b)
		prev = int(o.ID)
	}

	if len(m.Payload) > 0 {
		buf.Write([]byte{PayloadMarker})
	}

	buf.Write(m.Payload)

	return buf.Bytes(), nil
}

// ParseMessage extracts the Message from the given input.
func ParseMessage(data []byte) (Message, error) {
	rv := Message{}
	return rv, rv.UnmarshalBinary(data)
}

// UnmarshalBinary parses the given binary slice as a Message.
func (m *Message) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return ErrShortPacket
	}

	if data[0]>>6 != 1 {
		return ErrInvalidVersion
	}

	m.Type = COAPType((data[0] >> 4) & 0x3)
	tokenLen := int(data[0] & 0xf)
	if tokenLen > 8 {
		return ErrInvalidTokenLen
	}

	m.Code = COAPCode(data[1])
	m.MessageID = binary.BigEndian.Uint16(data[2:4])

	if tokenLen > 0 {
		m.Token = make([]byte, tokenLen)
	}
	if len(data) < 4+tokenLen {
		return ErrTruncated
	}
	copy(m.Token, data[4:4+tokenLen])
	b := data[4+tokenLen:]
	prev := 0

	parseExtOpt := func(opt int) (int, error) {
		switch opt {
		case extoptByteCode:
			if len(b) < 1 {
				return -1, ErrTruncated
			}
			opt = int(b[0]) + extoptByteAddend
			b = b[1:]
		case extoptWordCode:
			if len(b) < 2 {
				return -1, ErrTruncated
			}
			opt = int(binary.BigEndian.Uint16(b[:2])) + extoptWordAddend
			b = b[2:]
		}
		return opt, nil
	}

	for len(b) > 0 {
		if b[0] == PayloadMarker {
			b = b[1:]
			break
		}

		delta := int(b[0] >> 4)
		length := int(b[0] & 0x0f)

		if delta == extoptError || length == extoptError {
			return ErrInvalidOptionMarker
		}

		b = b[1:]

		delta, err := parseExtOpt(delta)
		if err != nil {
			return err
		}
		length, err = parseExtOpt(length)
		if err != nil {
			return err
		}

		if len(b) < length {
			return ErrTruncated
		}

		oid := OptionID(prev + delta)
		opval := parseOptionValue(oid, b[:length])
		b = b[length:]
		prev = int(oid)

		if opval != nil {
			m.opts = append(m.opts, option{ID: oid, Value: opval})
		}
	}
	m.Payload = b
	return nil
}
