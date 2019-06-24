package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
)

var openFileHook = os.OpenFile

func getEncoding(charset string) (encoding.Encoding, error) {
	return htmlindex.Get(charset)
}

type Opener struct {
	// Name은 열 엑셀 파일 이름입니다.
	Name string

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

	// OpenFlag는 파일을 열 때 사용할 설정을 지정합니다.
	OpenFlag int

	// OpenMode는 파일을 열거나 생성할 때 사용할 권한을 설정합니다. Mode 값을
	// 명시하지 않았다면 os.ModePerm 값을 기본으로 사용합니다.
	OpenMode os.FileMode
}

func (o *Opener) getEncoding() (encoding.Encoding, error) {
	if o.Encoding != nil {
		return o.Encoding, nil
	}
	if o.Charset != "" {
		return getEncoding(o.Charset)
	}
	return nil, nil
}

func (o *Opener) openFile() (*os.File, error) {
	return openFileHook(o.Name, o.OpenFlag, o.OpenMode)
}

// Open은 명시된 설정과 함께 엑셀 파일을 엽니다. 반환되는 *os.File 객체는
// 자동으로 닫히지 않으며 호출자가 닫을 책임을 지닙니다.
func (o Opener) Writer() (*Writer, *os.File, error) {
	enc, err := o.getEncoding()
	if err != nil {
		return nil, nil, err
	}
	raw, err := o.openFile()
	if err != nil {
		return nil, nil, err
	}
	w := (io.Writer)(raw)
	if enc != nil {
		w = enc.NewEncoder().Writer(w)
	}
	return NewWriter(csv.NewWriter(w)), raw, nil
}

// TODO
func (o Opener) Reader() (*Reader, *os.File, error) {
	enc, err := o.getEncoding()
	if err != nil {
		return nil, nil, err
	}
	raw, err := o.openFile()
	if err != nil {
		return nil, nil, err
	}
	r := (io.Reader)(raw)
	if enc != nil {
		r = enc.NewDecoder().Reader(r)
	}
	return NewReader(csv.NewReader(r)), raw, nil
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
	Writer     *csv.Writer
	recordPool sync.Pool
}

func NewWriter(w *csv.Writer) *Writer {
	return &Writer{
		Writer: w,
	}
}

func (w *Writer) borrowRecord(size int) []string {
	record, ok := w.recordPool.Get().([]string)
	if cap(record) < size {
		const e = 1
		if ok {
			w.recordPool.Put(record)
		}
		record = make([]string, size+e)
	}
	return record[:size]
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

func stringifyField(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Invalid:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func (w *Writer) Write(record interface{}) error {
	typ := reflect.TypeOf(record)
	if typ.Kind() != reflect.Slice {
		panic("record is not a slice")
	}
	v := reflect.ValueOf(record)

	// special case: equivalent to []string
	if typ.Elem().Kind() == reflect.String {
		err := w.Writer.Write(unsafeStrings(v))
		runtime.KeepAlive(record)
		return err
	}

	src := w.borrowRecord(v.Len())
	for i := 0; i < v.Len(); i++ {
		src[i] = stringifyField(v.Index(i))
	}
	err := w.Writer.Write(src)
	w.recordPool.Put(src)
	return err
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
	w.Writer.Flush()
	return w.Writer.Error()
}

// TODO
type Reader struct {
	Reader *csv.Reader
}

func NewReader(r *csv.Reader) *Reader {
	return &Reader{
		Reader: r,
	}
}
