package main

import (
	"fmt"
	"os"
	"net"
	"regexp"
	"sort"
	"strings"

	"github.com/wfarr/termtable"
)

func connect() (net.Conn) {
	conn, err := net.Dial("unix", "/var/lib/nagios/rw/nagios.qh")
	if err != nil {
		panic(err.Error())
	}
    return conn
}

func main() {
	// establish connection to our socket, for both reads and writes
    conn := connect()
	defer conn.Close()

	// suss out what command we actually wish to run
	if len(os.Args) == 0 {
		send_command(conn, "help")
	}

	switch os.Args[1] {
    case "status", "check":
        output := ""
        for _,host := range os.Args[2:] {
            send_command(conn, os.Args[1] + " " + host)
            output += read_results(conn)
            conn.Close()
            conn = connect()
        }
        print_table(output)
        return
	case "help", "acknowledge", "unacknowledge", "disable_notifications", "enable_notifications", "downtime", "problems", "muted", "stats":
		send_command(conn, strings.Join(os.Args[1:], " "))
	case "ack":
		send_command(conn, "acknowledge" + " " + strings.Join(os.Args[2:], " "))
	case "unack":
		send_command(conn, "unacknowledge" + " " + strings.Join(os.Args[2:], " "))
	case "mute":
		send_command(conn, "disable_notifications" + " " + strings.Join(os.Args[2:], " "))
	case "unmute":
		send_command(conn, "enable_notifications" + " " + strings.Join(os.Args[2:], " "))
	default:
		send_command(conn, "help")
	}

	output := read_results(conn)
    print_table(output)
}

func print_table(output string) {
	lines := strings.Split(output, "\n")
	table_it, _ := regexp.MatchString(";", output)

	if len(lines[1:]) > 0 && table_it {
		sort.Sort(sort.StringSlice(lines))

		t := termtable.NewTable(&termtable.TableOptions{Padding: 1, Header: []string{"Service", "Status", "Details"}, MaxColWidth: 72,})

		for _, line := range lines[1:] {
			parts := [3]string{"", "", ""}
			split := strings.Split(line, ";")

                        for i := 0; i < 3; i++ {
				var part string

				if i == 2 {
					part = strings.Join(split[2:], "")
				} else {
					part = split[i]
				}

				parts[i] = part

				if parts[i] == "" {
					parts[i] = "wat"
				}
			}

			row := []string{ parts[0], parts[1], parts[2] }
			t.AddRow(row)
		}

		fmt.Println(t.Render())
	} else {
		fmt.Println(strings.Join(lines[0:len(lines)-1], "\n"))
	}
}

func send_command(conn net.Conn, cmd string) {
	_, err := conn.Write([]byte(fmt.Sprintf("#nagioseasier %s\000", cmd)))
	if err != nil {
		panic(err.Error())
	}

}

func read_results(conn net.Conn) (output string) {
	buf := make([]byte, 4096)
	for ;; {
		n, err := conn.Read(buf[:])

		if err != nil {
			return scrub(output)
		}

		if n == 0 {
			fmt.Println("Connection closed by socket")
			return scrub(output)
		}

		output = output + string(buf[0:n])
	}
}

func scrub(input string) (output string) {
	// get rid of pesky null chars
	output = strings.Replace(input, "\000", "", -1)

	// get rid of fake newlines, lol
	output = strings.Replace(output, "\\n", "", -1)

	// chomp off trailing newlines
	output = strings.Trim(output, "\n")

	// trim spaces
	output = strings.TrimSpace(output)

	return
}
