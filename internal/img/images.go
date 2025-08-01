package img

type ImageBackend interface {
	SymbolsOnly() bool
	Render(path string, width, height int, symbols bool) (string, error)
}

func Get(backend string) ImageBackend {
	switch backend {
	case "docs":
		return NewDocsBackend()
	case "chafa":
		fallthrough
	default:
		return NewChafaBackend()
	}
}
