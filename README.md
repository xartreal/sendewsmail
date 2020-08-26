# sendexmail
Библиотека для отправки писем через корпоративный Exchange

Пример использования:
```
package main

import (
	"fmt"
	"io/ioutil"
	"github.com/xartreal/sendewsmail"
	"os"
	"path/filepath"
)

func main() {
	// usage: prog tomail file
	if len(os.Args) != 3 {
		fmt.Printf("usage: send tomail file\n")
		os.Exit(2)
	}
	tomail := os.Args[1] //recipient mail address
	fname := os.Args[2]
	fmt.Printf("hello\n")
	mm := sendexmail.InitMailer("alice@example.com", "https://mail.example.com/ews/Exchange.asmx", "alice.e", "12345", true)
	mm.InitLetters(5, "", false)
	defer mm.ClearFiles()
	subj := "Это тестовое письмо"
	fdata, _ := ioutil.ReadFile(fname)
	mm.AddAttachment(filepath.Base(fname), fdata)
	mm.SendMail(tomail, "Dear fiend", subj, "Hello!\nЭто письмо с файлом")
}

```
