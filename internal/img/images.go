package img

type ImageBackend interface {
	Render(path string, width, height int32) (string, error)
}

func Get(backend string) ImageBackend {
	switch backend {
	case "chafa":
		fallthrough
	default:
		return chafaBackend{}
	}
}
