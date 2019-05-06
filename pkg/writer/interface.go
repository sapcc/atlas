package writer

type Writer interface {
	GetData(string) (string, error)
	Write(name, data string) error
}
