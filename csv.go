package csv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strconv"
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
	if o.OpenFlag&os.O_RDONLY == 0 && o.OpenFlag&os.O_RDWR == 0 {
		o.OpenFlag |= os.O_RDONLY
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

func (o Opener) Reader() (*Reader, *os.File, error) {
	enc, err := o.getEncoding()
	if err != nil {
		return nil, nil, err
	}
	if o.OpenFlag&os.O_WRONLY == 0 && o.OpenFlag&os.O_RDWR == 0 {
		o.OpenFlag |= os.O_WRONLY
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
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return ""
		}
		return stringifyField(v.Elem())
	}
	return fmt.Sprint(v)
}

func (w *Writer) Write(record interface{}) error {
	if record == nil {
		return w.Writer.Write(nil)
	}

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

type Reader struct {
	Reader *csv.Reader
	record []string
	err    error
}

func NewReader(r *csv.Reader) *Reader {
	return &Reader{
		Reader: r,
	}
}

func (r *Reader) Skip() error {
	_, err := r.Reader.Read()
	if err, ok := err.(*csv.ParseError); ok {
		if err.Err == csv.ErrFieldCount {
			return nil
		}
	}
	return err
}

func (r *Reader) Err() error {
	if r.err == io.EOF {
		return nil
	}
	return r.err
}

func (r *Reader) Next() bool {
	r.record, r.err = r.Reader.Read()
	return r.err == nil
}

func (r *Reader) Record() []string {
	return r.record
}

func (r *Reader) Scan(dest ...interface{}) error {
	for i, dst := range dest {
		var val string
		if i < len(r.record) {
			val = r.record[i]
		}
		err := convertAssign(dst, val)
		if err != nil {
			return err
		}
	}
	return nil
}

var errNilPtr = errors.New("destination pointer is nil")

func convertAssign(dst interface{}, val string) error {
	switch dst := dst.(type) {
	case *string:
		if dst == nil {
			return errNilPtr
		}
		*dst = val
		return nil
	case *[]byte:
		if dst == nil {
			return errNilPtr
		}
		*dst = []byte(val)
		return nil
	}
	pv := reflect.ValueOf(dst)
	if pv.Kind() != reflect.Ptr {
		return errors.New("destination is not a pointer")
	}
	if pv.IsNil() {
		return errNilPtr
	}
	return convertAssignReflect(pv.Type(), pv.Elem(), val)
}

func convertAssignReflect(dstType reflect.Type, v reflect.Value, val string) error {
	// assert: v.CanSet()

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return convertAssignReflect(dstType, v.Elem(), val)
	case reflect.Interface:
		if v.Elem().CanSet() {
			return convertAssignReflect(dstType, v.Elem(), val)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i64, err := strconv.ParseInt(val, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u64, err := strconv.ParseUint(val, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(u64)
		return nil
	case reflect.Float32, reflect.Float64:
		f64, err := strconv.ParseFloat(val, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f64)
		return nil
	case reflect.String:
		v.SetString(val)
		return nil
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			v.Set(reflect.ValueOf([]byte(val)))
		}
	}

	return fmt.Errorf("unsupported Scan, storing into type %s",
		dstType.String())
}
