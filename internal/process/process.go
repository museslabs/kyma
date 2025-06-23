package process

type PreProcessor interface {
	Pre(content string, themeName string, animating bool) (string, error)
}
