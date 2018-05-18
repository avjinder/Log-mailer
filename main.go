package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func main() {
	var createJSON bool
	var jsonfile string
	flag.BoolVar(&createJSON, "generate", false, "Generate configuration file.")
	flag.StringVar(&jsonfile, "conf", "configuration.json", "Path to configuration file.")
	flag.Parse()
	if createJSON {
		reader := bufio.NewReader(os.Stdin)
		if r := ask("Location to store configuration file (default: ./" + jsonfile + "): "); strings.TrimSpace(r) != "" {
			jsonfile = r
		}
		if _, err := os.Stat(jsonfile); err == nil {
			fmt.Print("Configuration file exists, overwrite? (y/n): ")
			if r, _ := reader.ReadString('\n'); strings.ToLower(strings.TrimSpace(r)) != "y" {
				return
			}
			err := os.Remove(jsonfile)
			if err != nil {
				log.Println(err)
			}
		}
		var (
			fromName  = ask("Enter \"From\" name:\t")
			fromEmail = ask("Enter \"From\" email:\t")
			toName    = ask("Enter \"To\" name:\t")
			toEmail   = ask("Enter \"To\" email:\t")
			subject   = ask("Enter subject:\t\t")
			server    = ask("Enter SMTP server:\t")
			port      = ask("Port:\t\t\t")
			user      = ask("Username:\t\t")
			pass      = ask("Password:\t\t")
			logFile   = ask("Location of logs:\t")
			interval  = ask("Interval:\t\t")
			reset     = ask("Reset log file? (y/n):\t")
		)
		if reset != "y" {
			reset = "false"
		} else {
			reset = "true"
		}
		f, err := os.OpenFile(jsonfile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(f,
			"{\n\t\"from\": {\n\t\t\"name\": \"%s\",\n\t\t\"email\": \"%s\"\n\t},\n\t\"to\": {\n\t\t\"name\""+
				": \"%s\",\n\t\t\"email\": \"%s\"\n\t},\n\t\"subject\": \"%s\",\n\t\"server\": \"%s\",\n\t\""+
				"port\": \"%s\",\n\t\"credentials\": {\n\t\t\"user\": \"%s\",\n\t\t\"password\": \"%s\"\n\t}"+
				",\n\t\"logs\": \"%s\",\n\t\"interval\": \"%s\",\n\t\"reset\": \"%s\"\n}",
			fromName, fromEmail, toName, toEmail, subject, server, port, user, pass, logFile, interval, reset)
		f.Close()
		fmt.Printf("Configuration file generated: %s.\n", jsonfile)
		return
	}
	if _, err := os.Stat(jsonfile); err != nil {
		fmt.Printf("Unable to find configuration file (%s).\n", jsonfile)
		return
	}
	data, err := ioutil.ReadFile(jsonfile)
	if err != nil {
		log.Println(err)
	}
	var (
		from     = fmt.Sprintf(`"%s" <%s>`, get(data, "from.name"), get(data, "from.email"))
		to       = fmt.Sprintf(`"%s" <%s>`, get(data, "to.name"), get(data, "to.email"))
		server   = get(data, "server")
		port     = get(data, "port")
		user     = get(data, "credentials.user")
		pass     = get(data, "credentials.password")
		sub      = get(data, "subject")
		logs     = get(data, "logs")
		interval = get(data, "interval")
		reset, e = strconv.ParseBool(get(data, "reset"))
		message  = ""
	)
	if e != nil {
		log.Print(e)
		return
	}
	if interval[0] == '+' || interval[0] == '-' {
		interval = strings.Replace(interval, string(interval[0]), "", -1)
	}
	repeat(func() {
		if i, _ := os.Stat(logs); !(i.Size() > 0) {
			log.Printf("\n%s\n\n", "Log file is empty.")
			return
		}
		message = ""
		headers := make(map[string]string)
		headers["From"] = from
		headers["To"] = to
		headers["Subject"] = sub
		headers["MIME-version"] = "1.0;\nContent-Type: text/html; charset=\"UTF-8\";"
		for title, data := range headers {
			message += fmt.Sprintf("%s: %s\r\n", title, data)
		}
		message += "\r\n"
		file, err := os.Open(logs)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		message += "<div style=\"font-family: monospace;background: #ecf0f1;padding: 20px;border-radius: 9px;font-size: 150%;margin: 30px;\">"
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			message += scanner.Text() + "<br>"
		}
		message += "<br><br>Generated by <a href=\"https://github.com/muhammadmuzzammil1998/Log-mailer\">Log Mailer</a> on " + time.Now().Format(time.RFC1123Z) + "</div>"
		err = smtp.SendMail(
			server+":"+port,
			smtp.PlainAuth("", user, pass, server),
			headers["From"],
			[]string{headers["To"]},
			[]byte(message),
		)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("\n%s\n\n", message)
		if reset {
			os.Remove(logs)
			os.Create(logs)
		}
	}, interval)
}
func repeat(f func(), interval string) {
	f()
	d, err := time.ParseDuration(interval)
	if err != nil {
		log.Println(err)
	}
	for _ = range time.Tick(d) {
		f()
	}
}
func get(data []byte, path string) string {
	return gjson.Get(fmt.Sprintf("%s", data), path).String()
}
func ask(s string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(s)
	r, _, _ := reader.ReadLine()
	return string(r)
}
