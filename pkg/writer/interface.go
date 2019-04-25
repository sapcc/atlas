package writer

type Writer interface {
	GetData() (string, error)
	Write(string) error
}
