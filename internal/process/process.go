package process

type PreProcessor interface {
	Pre(content string, animating bool) (string, error)
}

type PostProcessor interface {
	Post(content string) (string, error)
}

type Processor interface {
	PreProcessor
	PostProcessor
}
