package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

type Opener struct {
	// Name은 열 엑셀 파일 이름입니다.
	Name string

	// ColumnNames는 엑셀 파일 생성 시 파일의 첫 행에 쓰여집니다.
	ColumnNames interface{}

	// Charset은 데이터 저장에 사용할 인코딩 이름을 설정합니다. 인코딩을
	// 설정하지 않는다면 기본값인 utf-8을 사용합니다.
	// http://www.w3.org/TR/encoding 목록에 등록된 인코딩이 지원됩니다.
	// 한국어 사용을 원하신다면 euc-kr 혹은 korean 인코딩을 입력하면
	// 됩니다.
	// 지원되지 않는 인코딩 사용을 원하신다면 아래 Encoding 설정을
	// 참고하십시오.
	// 잘못된 인코딩 이름이 입력된다면 Open 함수는 실패할 수 있습니다.
	Charset string

	// Encoding은 데이터 저장에 사용할 인코딩입니다. Encoding은 Charset보다
	// 우선순위가 높습니다. Charset이 지원하지 않는 인코딩을 설정할 때
	// 유용합니다.
	Encoding encoding.Encoding

	// Mode는 파일을 열거나 생성할 때 사용할 권한을 설정합니다. Mode 값을
	// 명시하지 않았다면 os.ModePerm 값을 기본으로 사용합니다.
	Mode os.FileMode

	// Create가 설정되고 여는 파일이 없다면 파일을 새로 생성합니다.
	Create bool

	// Append가 설정된다면 데이터 쓰기 시 기존 파일을 덮어씌우지 않고 아닌
	// 파일 마지막에 추가합니다.
	Append bool

	// Sync가 설정된다면 데이터 쓰기를 버퍼를 거치지 않고 바로
	// 파일시스템으로 보냅니다.
	Sync bool
}

// Open은 명시된 설정과 함께 엑셀 파일을 엽니다. 반환되는 *os.File 객체는
// 자동으로 닫히지 않으며 호출자가 닫을 책임을 지닙니다.
func (o Opener) Open() (*Writer, *os.File, error) {
	openFlag := os.O_WRONLY
	if o.Create {
		openFlag |= os.O_CREATE
	}
	if o.Append {
		openFlag |= os.O_APPEND
	}
	if o.Sync {
		openFlag |= os.O_SYNC
	}
	openMode := o.Mode
	if openMode == 0 {
		openMode = os.ModePerm
	}
	if o.Charset != "" && o.Encoding == nil {
		enc, err := htmlindex.Get(o.Charset)
		if err != nil {
			return nil, nil, err
		}
		o.Encoding = enc
	}
	raw, err := os.OpenFile(o.Name, openFlag, openMode)
	if err != nil {
		return nil, nil, err
	}
	w := NewWriter(raw)
	w.Encoding = o.Encoding

	if o.ColumnNames != nil && isEmptyFile(raw) {
		w.Write(o.ColumnNames)
	}
	return w, nil, nil
}

func isEmptyFile(raw *os.File) bool {
	is, err := IsEmptyFile(raw)
	return err == nil && is
}

func IsEmptyFile(raw interface {
	Stat() (os.FileInfo, error)
}) (bool, error) {
	st, err := raw.Stat()
	return err == nil && st.Size() == 0, err
}

type Writer struct {
	// Encoding은 데이터 쓰기 시 사용하는 인코딩입니다. 지정하지 않는다면
	// 기본값인 utf-8이 사용됩니다. Encoding 필드는 Write, WriteAll, Fields
	// 함수를 호출한 이후엔 읽기 전용이 됩니다.
	Encoding encoding.Encoding

	raw io.Writer
	w   *csv.Writer

	enc        *encoding.Encoder
	bufferPool sync.Pool
	recordPool sync.Pool
}

func NewWriter(raw io.Writer) *Writer {
	return &Writer{
		raw: raw,
		w:   csv.NewWriter(raw),
	}
}

type borrowing struct {
	src      []string
	borrowed bool
}

func (w *Writer) encode(b borrowing) (borrowing, error) {
	if w.isUTF8() {
		return b, nil
	}
	dst := b.src
	if !b.borrowed {
		dst = w.borrowRecord(len(dst))
	}
	for i, s := range b.src {
		var err error
		dst[i], err = w.encodeString(s)
		if err != nil {
			w.recordPool.Put(dst)
			return borrowing{}, err
		}
	}
	return borrowing{dst, true}, nil
}

// assert: !w.isUTF8()
func (w *Writer) encodeString(s string) (string, error) {
	if w.enc == nil {
		w.enc = w.Encoding.NewEncoder()
	}
	buf := w.getBuf()
	buf.Reset()
	defer w.bufferPool.Put(buf)
	err := flood(transform.NewWriter(buf, w.enc.Transformer), unsafeBytes(s))
	runtime.KeepAlive(s)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func flood(w io.WriteCloser, p []byte) error {
	_, err := w.Write(p)
	if err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

// unsafe
func unsafeBytes(s string) []byte {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&s))
	h.Cap = h.Len
	return *(*[]byte)(unsafe.Pointer(&h))
}

func (w *Writer) getBuf() *bytes.Buffer {
	buf, _ := w.bufferPool.Get().(*bytes.Buffer)
	if buf == nil {
		buf = new(bytes.Buffer)
		buf.Grow(2048)
	}
	return buf
}

func (w *Writer) borrowRecord(size int) []string {
	record, _ := w.recordPool.Get().([]string)
	if size < cap(record) {
		const e = 1
		w.recordPool.Put(record)
		record = make([]string, size+e)
	}
	return record
}

func (w *Writer) reclaimRecord(record []string) {
	w.recordPool.Put(record)
}

func (w *Writer) reclaim(b borrowing) {
	if b.borrowed {
		w.reclaimRecord(b.src)
	}
}

func (w *Writer) isUTF8() bool {
	if w.Encoding == nil {
		return true
	}
	name, err := htmlindex.Name(w.Encoding)
	if err != nil {
		return false
	}
	return strings.EqualFold(name, "utf-8")
}

// unsafe
func unsafeStrings(v reflect.Value) []string {
	repr := reflect.SliceHeader{
		v.Pointer(),
		v.Len(),
		v.Cap(),
	}
	return *(*[]string)(unsafe.Pointer(&repr))
}

func (w *Writer) stringify(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

// row []string
// row []interface{}
func (w *Writer) Write(record interface{}) error {
	typ := reflect.TypeOf(record)
	if typ.Kind() != reflect.Slice {
		panic("record is not a slice")
	}
	v := reflect.ValueOf(record)

	// special case: equivalent to []string
	if typ.Elem().Kind() == reflect.String {
		b, err := w.encode(borrowing{src: unsafeStrings(v)})
		if err != nil {
			return err
		}
		err = w.w.Write(b.src)
		w.reclaim(b)
		runtime.KeepAlive(record)
		return err
	}

	dst := w.borrowRecord(v.Len())
	for i := 0; i < v.Len(); i++ {
		dst[i] = w.stringify(v.Index(i))
	}
	defer w.reclaimRecord(dst)
	b, err := w.encode(borrowing{dst, true})
	if err != nil {
		return err
	}
	return w.w.Write(b.src)
}

func (w *Writer) WriteAll(records []interface{}) error {
	for _, record := range records {
		err := w.Write(record)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) Fields(record ...interface{}) error {
	return w.Write(record)
}

func (w *Writer) Flush() error {
	w.w.Flush()
	return w.w.Error()
}
