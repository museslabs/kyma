package process

type PreProcessor interface {
	Pre(content string) (string, error)
}

type PostProcessor interface {
	Post(content string) (string, error)
}

type Processor interface {
	PreProcessor
	PostProcessor
}
