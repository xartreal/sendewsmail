// mailer-vars
package sendewsmail

// sender configuration
type TConfig struct {
	from      string
	endpoint  string
	username  string
	userpass  string
	debugmode bool
}

// letter configuration
type TLettersConfig struct {
	letterpause int    // интервал между письмами (используется в случае рассылки)
	signature   string // подпись снизу письма
	rcenable    bool   //receipt enable (требовать подтверждения доставки)
}

// attachment list and storage
type TAttachList struct {
	Filenames   []string
	FileContent map[string]string
}

// mailer configuration
type Mailer struct {
	Config        TConfig
	LettersConfig TLettersConfig
	AttachList    TAttachList
}

//message data (internal)
var (
	message string
	title   string
)
