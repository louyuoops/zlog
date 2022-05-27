package zlog

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var (
	bufferpool = buffer.NewPool()
)

// For JSON-escaping; see textEncoder.safeAddString below.
const _hex = "0123456789abcdef"

var (
	_pool = buffer.NewPool()
	// Get retrieves a buffer from the pool, creating one if necessary.
	Get = _pool.Get
)

var _textPool = sync.Pool{New: func() interface{} {
	return &textEncoder{}
}}

func gettextEncoder() *textEncoder {
	return _textPool.Get().(*textEncoder)
}

func getTextEncoder(config zapcore.EncoderConfig) (zapcore.Encoder, error) {
	return NewtextEncoder(config), nil
}

func puttextEncoder(enc *textEncoder) {
	if enc.reflectBuf != nil {
		enc.reflectBuf.Free()
	}
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.spaced = false
	enc.openNamespaces = 0
	enc.reflectBuf = nil
	enc.reflectEnc = nil
	_textPool.Put(enc)
}

type textEncoder struct {
	*zapcore.EncoderConfig
	buf            *buffer.Buffer
	spaced         bool // include spaces after colons and commas
	openNamespaces int

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc zapcore.ReflectedEncoder
}

// NewtextEncoder creates a fast, low-allocation JSON encoder. The encoder
// appropriately escapes all field keys and values.
//
// Note that the encoder doesn't deduplicate keys, so it's possible to produce
// a message like
//   {"foo":"bar","foo":"baz"}
// This is permitted by the JSON specification, but not encouraged. Many
// libraries will ignore duplicate key-value pairs (typically keeping the last
// pair) when unmarshaling, but users should attempt to avoid adding duplicate
// keys.
func NewtextEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return newtextEncoder(cfg, false)
}

func defaultReflectedEncoder(w io.Writer) zapcore.ReflectedEncoder {
	enc := json.NewEncoder(w)
	// For consistency with our custom JSON encoder.
	enc.SetEscapeHTML(false)
	return enc
}

func newtextEncoder(cfg zapcore.EncoderConfig, spaced bool) *textEncoder {
	if cfg.SkipLineEnding {
		cfg.LineEnding = ""
	} else if cfg.LineEnding == "" {
		cfg.LineEnding = zapcore.DefaultLineEnding
	}

	// If no zapcore.EncoderConfig.NewReflectedEncoder is provided by the user, then use default
	if cfg.NewReflectedEncoder == nil {
		cfg.NewReflectedEncoder = defaultReflectedEncoder
	}

	return &textEncoder{
		EncoderConfig: &cfg,
		buf:           bufferpool.Get(),
		spaced:        spaced,
	}
}

func (enc *textEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	enc.addKey(key)
	return enc.AppendArray(arr)
}

func (enc *textEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	enc.addKey(key)
	return enc.AppendObject(obj)
}

func (enc *textEncoder) AddBinary(key string, val []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (enc *textEncoder) AddByteString(key string, val []byte) {
	enc.addKey(key)
	enc.AppendByteString(val)
}

func (enc *textEncoder) AddBool(key string, val bool) {
	enc.addKey(key)
	enc.AppendBool(val)
}

func (enc *textEncoder) AddComplex128(key string, val complex128) {
	enc.addKey(key)
	enc.AppendComplex128(val)
}

func (enc *textEncoder) AddComplex64(key string, val complex64) {
	enc.addKey(key)
	enc.AppendComplex64(val)
}

func (enc *textEncoder) AddDuration(key string, val time.Duration) {
	enc.addKey(key)
	enc.AppendDuration(val)
}

func (enc *textEncoder) AddFloat64(key string, val float64) {
	enc.addKey(key)
	enc.AppendFloat64(val)
}

func (enc *textEncoder) AddFloat32(key string, val float32) {
	enc.addKey(key)
	enc.AppendFloat32(val)
}

func (enc *textEncoder) AddInt64(key string, val int64) {
	enc.addKey(key)
	enc.AppendInt64(val)
}

func (enc *textEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = bufferpool.Get()
		enc.reflectEnc = enc.NewReflectedEncoder(enc.reflectBuf)
	} else {
		enc.reflectBuf.Reset()
	}
}

var nullLiteralBytes = []byte("null")

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (enc *textEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	if err := enc.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	enc.reflectBuf.TrimNewline()
	return enc.reflectBuf.Bytes(), nil
}

func (enc *textEncoder) AddReflected(key string, obj interface{}) error {
	valueBytes, err := enc.encodeReflected(obj)
	if err != nil {
		return err
	}
	enc.addKey(key)
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *textEncoder) OpenNamespace(key string) {
	enc.addKey(key)
	enc.buf.AppendByte('{')
	enc.openNamespaces++
}

func (enc *textEncoder) AddString(key, val string) {
	if key == "G_SERV_NAME" {
		enc.buf.AppendByte('[')
		enc.buf.AppendString("G_SERV_NAME:")
		enc.buf.AppendString(val)
		enc.buf.AppendByte(']')
		return
	}
	enc.addKey(key)
	enc.AppendString(val)
}

func (enc *textEncoder) AddTime(key string, val time.Time) {
	enc.addKey(key)
	enc.AppendTime(val)
}

func (enc *textEncoder) AddUint64(key string, val uint64) {
	enc.addKey(key)
	enc.AppendUint64(val)
}

func (enc *textEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	enc.addElementSeparator()
	enc.buf.AppendByte('[')
	err := arr.MarshalLogArray(enc)
	enc.buf.AppendByte(']')
	return err
}

func (enc *textEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	// Close ONLY new openNamespaces that are created during
	// AppendObject().
	old := enc.openNamespaces
	enc.openNamespaces = 0
	enc.addElementSeparator()
	enc.buf.AppendByte('{')
	err := obj.MarshalLogObject(enc)
	enc.buf.AppendByte('}')
	enc.closeOpenNamespaces()
	enc.openNamespaces = old
	return err
}

func (enc *textEncoder) AppendBool(val bool) {
	enc.addElementSeparator()
	enc.buf.AppendBool(val)
}

func (enc *textEncoder) AppendByteString(val []byte) {
	enc.addElementSeparator()
	// enc.buf.AppendByte('"')
	enc.safeAddByteString(val)
	// enc.buf.AppendByte('"')
}

// appendComplex appends the encoded form of the provided complex128 value.
// precision specifies the encoding precision for the real and imaginary
// components of the complex number.
func (enc *textEncoder) appendComplex(val complex128, precision int) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, precision)
	// If imaginary part is less than 0, minus (-) sign is added by default
	// by AppendFloat.
	if i >= 0 {
		enc.buf.AppendByte('+')
	}
	enc.buf.AppendFloat(i, precision)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

func (enc *textEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

func (enc *textEncoder) AppendInt64(val int64) {
	enc.addElementSeparator()
	enc.buf.AppendInt(val)
}

func (enc *textEncoder) AppendReflected(val interface{}) error {
	valueBytes, err := enc.encodeReflected(val)
	if err != nil {
		return err
	}
	enc.addElementSeparator()
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *textEncoder) AppendString(val string) {
	enc.addElementSeparator()
	// enc.buf.AppendByte('"')
	enc.safeAddString(val)
	// enc.buf.AppendByte('"')
}

func (enc *textEncoder) AppendTimeLayout(time time.Time, layout string) {
	enc.addElementSeparator()
	// enc.buf.AppendByte('"')
	enc.buf.AppendTime(time, layout)
	// enc.buf.AppendByte('"')
}

func (enc *textEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	if e := enc.EncodeTime; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

func (enc *textEncoder) AppendUint64(val uint64) {
	enc.addElementSeparator()
	enc.buf.AppendUint(val)
}

func (enc *textEncoder) AddInt(k string, v int)         { enc.AddInt64(k, int64(v)) }
func (enc *textEncoder) AddInt32(k string, v int32)     { enc.AddInt64(k, int64(v)) }
func (enc *textEncoder) AddInt16(k string, v int16)     { enc.AddInt64(k, int64(v)) }
func (enc *textEncoder) AddInt8(k string, v int8)       { enc.AddInt64(k, int64(v)) }
func (enc *textEncoder) AddUint(k string, v uint)       { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AddUint32(k string, v uint32)   { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AddUint16(k string, v uint16)   { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AddUint8(k string, v uint8)     { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AddUintptr(k string, v uintptr) { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AppendComplex64(v complex64)    { enc.appendComplex(complex128(v), 32) }
func (enc *textEncoder) AppendComplex128(v complex128)  { enc.appendComplex(complex128(v), 64) }
func (enc *textEncoder) AppendFloat64(v float64)        { enc.appendFloat(v, 64) }
func (enc *textEncoder) AppendFloat32(v float32)        { enc.appendFloat(float64(v), 32) }
func (enc *textEncoder) AppendInt(v int)                { enc.AppendInt64(int64(v)) }
func (enc *textEncoder) AppendInt32(v int32)            { enc.AppendInt64(int64(v)) }
func (enc *textEncoder) AppendInt16(v int16)            { enc.AppendInt64(int64(v)) }
func (enc *textEncoder) AppendInt8(v int8)              { enc.AppendInt64(int64(v)) }
func (enc *textEncoder) AppendUint(v uint)              { enc.AppendUint64(uint64(v)) }
func (enc *textEncoder) AppendUint32(v uint32)          { enc.AppendUint64(uint64(v)) }
func (enc *textEncoder) AppendUint16(v uint16)          { enc.AppendUint64(uint64(v)) }
func (enc *textEncoder) AppendUint8(v uint8)            { enc.AppendUint64(uint64(v)) }
func (enc *textEncoder) AppendUintptr(v uintptr)        { enc.AppendUint64(uint64(v)) }

func (enc *textEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *textEncoder) clone() *textEncoder {
	clone := gettextEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.spaced = enc.spaced
	clone.openNamespaces = enc.openNamespaces
	clone.buf = bufferpool.Get()
	return clone
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}

func (enc *textEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := enc.clone()

	// level header
	if final.EncodeLevel != nil {
		final.buf.AppendByte('[')
		final.EncodeLevel(ent.Level, final)
		final.buf.AppendByte(']')
	}

	// time part
	final.buf.AppendByte('[')
	final.AppendTime(ent.Time)
	// final.AddTime(final.TimeKey, ent.Time)
	final.buf.AppendByte(']')

	// service part
	// parse bytes to string for processing
	if enc.buf.Len() > 0 {
		final.addElementSeparator()
		final.buf.Write(enc.buf.Bytes())
	}

	if ent.Caller.Defined {
		// final.addKey(final.CallerKey)
		final.buf.AppendByte('[')
		cur := final.buf.Len()
		final.EncodeCaller(ent.Caller, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeCaller was a no-op. Fall back to strings to
			// keep output JSON valid.
			final.AppendString(ent.Caller.String())
		}
		final.buf.AppendByte(' ')
		final.buf.AppendString(ent.Caller.Function)
		final.buf.AppendByte(']')
	}
	final.buf.AppendByte(' ')

	if final.MessageKey != "" {
		final.addKey(enc.MessageKey)
		final.AppendString(ent.Message)
	}

	addFields(final, fields)
	if ent.Stack != "" && final.StacktraceKey != "" {
		final.AddString(final.StacktraceKey, ent.Stack)
	}
	final.buf.AppendString(final.LineEnding)

	ret := final.buf
	puttextEncoder(final)
	return ret, nil
}

func (enc *textEncoder) truncate() {
	enc.buf.Reset()
}

func (enc *textEncoder) closeOpenNamespaces() {
	for i := 0; i < enc.openNamespaces; i++ {
		enc.buf.AppendByte('}')
	}
	enc.openNamespaces = 0
}

func (enc *textEncoder) addKey(key string) {
	enc.addElementSeparator()
	// enc.buf.AppendByte('"')
	enc.safeAddString(key)
	// enc.buf.AppendByte('"')
	enc.buf.AppendByte('=')
	if enc.spaced {
		enc.buf.AppendByte(' ')
	}
}

func (enc *textEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', '=', ',', ' ', ']':
		return
	default:
		enc.buf.AppendByte('|')
		enc.buf.AppendByte('|')
		if enc.spaced {
			enc.buf.AppendByte(' ')
		}
	}
}

func (enc *textEncoder) appendFloat(val float64, bitSize int) {
	enc.addElementSeparator()
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *textEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *textEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (enc *textEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '"' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(_hex[b>>4])
		enc.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (enc *textEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}
