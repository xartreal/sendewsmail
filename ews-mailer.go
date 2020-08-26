// mailer
package sendewsmail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/vadimi/go-http-ntlm"
)

var TplCheckAccess = `
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"
        xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types"
        xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Header>
    <t:RequestServerVersion Version="Exchange2010_SP1" />
  </soap:Header>
  <soap:Body>
    <m:GetFolder>
      <m:FolderShape>
        <t:BaseShape>AllProperties</t:BaseShape>
      </m:FolderShape>
      <m:FolderIds>
        <t:DistinguishedFolderId Id="msgfolderroot" />
      </m:FolderIds>
    </m:GetFolder>
  </soap:Body>
</soap:Envelope>
`
var TplSendRC = `
          <t:IsDeliveryReceiptRequested>true</t:IsDeliveryReceiptRequested>
`

var TplSendText = `
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <soap:Body>
    <CreateItem MessageDisposition="%mailmode%" xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <SavedItemFolderId>
        <t:DistinguishedFolderId Id="sentitems" >
          <t:Mailbox>
           <t:EmailAddress>%mailfrom%</t:EmailAddress>
          </t:Mailbox>
        </t:DistinguishedFolderId>
      </SavedItemFolderId>
      <Items>
        <t:Message>
          <t:ItemClass>IPM.Note</t:ItemClass>
          <t:Subject>%mailsubj%</t:Subject>
          <t:Body BodyType="Text">%mailtext%</t:Body>
<t:Sender>
 <t:Mailbox>
  <t:EmailAddress>%mailfrom%</t:EmailAddress>
 </t:Mailbox>
</t:Sender>     

          <t:ToRecipients>
            <t:Mailbox>
              <t:Name>%toname%</t:Name>
              <t:EmailAddress>%mailto%</t:EmailAddress>
            </t:Mailbox>
          </t:ToRecipients>%rc%
          <t:IsRead>false</t:IsRead>
        </t:Message>
      </Items>
    </CreateItem>
  </soap:Body>
</soap:Envelope>
`
var TplSendAttach = `
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <soap:Body>
    <CreateAttachment xmlns="http://schemas.microsoft.com/exchange/services/2006/messages"
                      xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
      <ParentItemId Id="%mailid%" ChangeKey="%mailkey%"/>
      <Attachments>
    %mailfiles%
      </Attachments>
    </CreateAttachment>
  </soap:Body>
</soap:Envelope>
`
var TplAttachItem = `
        <t:FileAttachment>
          <t:Name>%filename%</t:Name>
          <t:Content>%filecontent%</t:Content>
        </t:FileAttachment>
`
var TplSendFinal = `
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema"
               xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <soap:Body>
    <SendItem xmlns="http://schemas.microsoft.com/exchange/services/2006/messages"
              SaveItemToFolder="true">
      <ItemIds>
        <t:ItemId Id="%mailid%" ChangeKey="%mailkey%" />
      </ItemIds>
    </SendItem>
  </soap:Body>
</soap:Envelope>
`

// internal: send request item to Exchange
func (m *Mailer) senditem(xmlin string, fname string) (string, error) {
	var resp *http.Response
	var err error
	if m.Config.debugmode {
		ioutil.WriteFile(fname+".log", []byte(xmlin), 0755)
	}
	req, err := http.NewRequest("POST", m.Config.endpoint, bytes.NewReader([]byte(xmlin)))
	if err != nil {
		log.Printf("Z0 error\n")
		return "", err
	}
	req.Header.Set("Content-Type", "text/xml")
	client := http.Client{
		Transport: &httpntlm.NtlmTransport{
			Domain:          "",
			User:            m.Config.username,
			Password:        m.Config.userpass,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	att := 0
	for {
		resp, err = client.Do(req)
		if err != nil {
			att++
			log.Printf("Attempt %d; Z1 error: %q\n", att, err.Error())
			if att > 4 {
				log.Printf("FATAL: Skipping via fatal response\n")
				return "", err
			}
			time.Sleep(1 * time.Minute)
		} else {
			break
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Status=%v", resp.StatusCode)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Z2 error\n")
		return "", err
	}
	bodyString := string(bodyBytes)

	return bodyString, nil
}

// Check credentials
func (m *Mailer) CheckCR() error {
	_, err := m.senditem(TplCheckAccess, "cr")
	return err
}

// Send textonly letter (one step)
func (m *Mailer) SendTextOnly(to string, nameto string, subj string, text string) {
	message := text + m.LettersConfig.signature
	mailxml := TplSendText
	mailxml = strings.Replace(mailxml, "%mailmode%", "SendAndSaveCopy", -1)
	mailxml = strings.Replace(mailxml, "%mailto%", to, -1)
	mailxml = strings.Replace(mailxml, "%toname%", nameto, -1)
	mailxml = strings.Replace(mailxml, "%mailfrom%", m.Config.from, -1)
	mailxml = strings.Replace(mailxml, "%mailsubj%", subj, -1)
	mailxml = strings.Replace(mailxml, "%mailtext%", message, -1)
	if m.LettersConfig.rcenable {
		mailxml = strings.Replace(mailxml, "%rc%", TplSendRC, -1)
	} else {
		mailxml = strings.Replace(mailxml, "%rc%", "", -1)
	}

	resp, err := m.senditem(mailxml, "r0")
	if err != nil {
		log.Printf("Error %q\n", err)
	}
	if m.Config.debugmode {
		ioutil.WriteFile("s0.log", []byte(resp), 0755)
	}
}

// Send text part of letter to Exchange (step 1 of 3)
// return item-id, item-key
func (m *Mailer) SendLetterStep(to string, nameto string, subj string, text string) (string, string) {
	message := text + m.LettersConfig.signature
	mailxml := TplSendText
	mailxml = strings.Replace(mailxml, "%mailmode%", "SaveOnly", -1)
	mailxml = strings.Replace(mailxml, "%mailto%", to, -1)
	mailxml = strings.Replace(mailxml, "%toname%", nameto, -1)
	mailxml = strings.Replace(mailxml, "%mailfrom%", m.Config.from, -1)
	mailxml = strings.Replace(mailxml, "%mailsubj%", subj, -1)
	mailxml = strings.Replace(mailxml, "%mailtext%", message, -1)
	if m.LettersConfig.rcenable {
		mailxml = strings.Replace(mailxml, "%rc%", TplSendRC, -1)
	} else {
		mailxml = strings.Replace(mailxml, "%rc%", "", -1)
	}
	resp, err := m.senditem(mailxml, "r1")
	if err != nil {
		log.Printf("Error %q\n", err)
	}
	if m.Config.debugmode {
		ioutil.WriteFile("s1.log", []byte(resp), 0755)
	}
	rx := regexp.MustCompile(`(?s)<t:ItemId\s+Id="(.*?)"\s+ChangeKey="(.*?)"`)
	tkn := rx.FindStringSubmatch(resp)
	if len(tkn) != 3 {
		log.Printf("Error tkn: %v\n", tkn)
		return "", ""
	}
	return tkn[1], tkn[2]
}

// Send attachment part of letter to Exchange (step 2 of 3)
// return item-key
func (m *Mailer) SendAttachStep(msgid string, msgkey string) string {
	mailxml := TplSendAttach
	mailxml = strings.Replace(mailxml, "%mailid%", msgid, -1)
	mailxml = strings.Replace(mailxml, "%mailkey%", msgkey, -1)
	fixml := ""
	for i := 0; i < len(m.AttachList.Filenames); i++ {
		tmpxml := TplAttachItem
		fname := m.AttachList.Filenames[i]
		tmpxml = strings.Replace(tmpxml, "%filename%", fname, -1)
		tmpxml = strings.Replace(tmpxml, "%filecontent%", m.AttachList.FileContent[fname], -1)
		fixml += tmpxml
	}
	mailxml = strings.Replace(mailxml, "%mailfiles%", fixml, -1)
	resp, err := m.senditem(mailxml, "r2")
	if err != nil {
		log.Printf("Error %q\n", err)
	}
	if m.Config.debugmode {
		ioutil.WriteFile("s2.log", []byte(resp), 0755)
	}
	rx := regexp.MustCompile(`(?s)RootItemChangeKey="(.+?)"`)
	tkn := rx.FindStringSubmatch(resp)
	if len(tkn) != 2 {
		log.Printf("Error tkn: %v\n", tkn)
		return ""
	}
	//mmid := tkn[1]
	return tkn[1]

}

// Send final part of letter to Exchange (step 3 of 3)
func (m *Mailer) SendLetterFinal(msgid string, msgkey string) {
	mailxml := TplSendFinal
	mailxml = strings.Replace(mailxml, "%mailid%", msgid, -1)
	mailxml = strings.Replace(mailxml, "%mailkey%", msgkey, -1)
	resp, err := m.senditem(mailxml, "r3")
	if err != nil {
		log.Printf("Error %q\n", err)
	}
	if m.Config.debugmode {
		ioutil.WriteFile("s3.log", []byte(resp), 0755)
	}
}

// Mailer initialization
func InitMailer(from string, endpoint string, username string, userpass string, debugenable bool) *Mailer {
	cc := TConfig{from, endpoint, username, userpass, debugenable}
	tp := TLettersConfig{0, "", false}
	ta := TAttachList{[]string{}, map[string]string{}}
	return &Mailer{cc, tp, ta}
}

// Init letter preferences
func (m *Mailer) InitLetters(letterpause int, signature string, rcenable bool) {
	tp := TLettersConfig{letterpause, signature, rcenable}
	m.LettersConfig = tp
}
