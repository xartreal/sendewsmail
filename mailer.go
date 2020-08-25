package sendexmail

import (
	"encoding/base64"
	"strings"
	"time"
)

// remove \n and \r
func unnr(s string) string {
	r := strings.Replace(s, "\n", "", -1)
	r = strings.Replace(r, "\r", "", -1)
	return r
}

// remove incorrect chars in name
func fixname(s string) string {
	r := strings.Replace(s, "<", `"`, -1)
	r = strings.Replace(s, ">", `"`, -1)
	return r
}

// Clear attachment list and storage
func (m *Mailer) ClearFiles() {
	m.AttachList.FileContent = map[string]string{}
	m.AttachList.Filenames = []string{}
}

// Add attachment to letter
func (m *Mailer) AddAttachment(filename string, filedata []byte) {
	m.AttachList.Filenames = append(m.AttachList.Filenames, filename)
	m.AttachList.FileContent[filename] = base64.StdEncoding.EncodeToString(filedata)
}

// Send letters
func (m *Mailer) SendMail(tomail string, toname string, title string, message string) {
	if len(m.AttachList.Filenames) == 0 {
		m.SendTextOnly(tomail, fixname(toname), unnr(title), message)
	} else {
		mmid, mmkey := m.SendLetterStep(tomail, fixname(toname), unnr(title), message)
		if (len(mmid) < 2) || (len(mmkey) < 2) {
			return
		}
		mmkey = m.SendAttachStep(mmid, mmkey)
		if len(mmkey) < 2 {
			return
		}
		m.SendLetterFinal(mmid, mmkey)
	}
	if m.LettersConfig.letterpause > 0 {
		for i := 0; i < m.LettersConfig.letterpause; i++ {
			time.Sleep(time.Second)
		}
	}
}
