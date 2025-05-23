package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// jsonOutput structure contains the data extracted from a
// "From:" string in an email data
// It contains the display name, email address and error state
type jsonOutput struct {
	Name  string `json:"display_name"`
	Email string `json:"addr_spec"`
	Error string `json:"error"`
}

// start of the application
//
// Returns exit status to the OS
func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s file.eml\n", os.Args[0])
		fmt.Printf("Usage: %s filename <for custom create test strings in a file>\n", os.Args[0])
		os.Exit(0)
	}

	//Run against a specific file containg all data from the header
	if strings.Contains(os.Args[1], ".eml") {
		senderInfo, err := parseFile(os.Args[1])
		if err != nil {
			os.Exit(1)
		}

		//display the data on the stdout - console in json format
		displayData(senderInfo, err)
	} else {
		//run tests from a external file, where
		//everyline is a specific "Form:" string
		doCustomFileTests(os.Args[1])
	}
}

// Parse the filename that is sent as a parameter to the application
//
// Returns an error if filename can not be opened, located the "Fron:" string and extract the email info
// or a valid map of display name and/or email
func parseFile(filename string) (map[string]string, error) {

	//Open the file that is passed from the command line as an argument and check it for error
	fd, err := os.Open(filename)
	if err != nil {
		fmt.Printf(err.Error())
		return nil, err
	}

	str, err := locateString(fd, "From:")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	str, err = checkForErrors(str)
	if err != nil {
		return nil, err
	}

	//extract the data from the "From:" string
	senderInfo, err := extractEmailInfo(str)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//close the opened file and check for error
	err = fd.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return senderInfo, nil
}

// output the json data extracted from the email to stdout
//
// Return void/noting
func displayData(senderInfo map[string]string, err error) {

	jsonOut := jsonOutput{Error: "null"}
	jsonOut.Name = senderInfo["display_name"]
	jsonOut.Email = senderInfo["addr_spec"]
	if err != nil {
		jsonOut.Error = err.Error()
	} else {
		jsonOut.Error = "null"
	}
	fmt.Printf(" %s\n", createJSONOutput(jsonOut))
}

// locate a string("From:") in a string
//
// Return nil if the string is not locate, or the line where the search string is found
func locateString(fd *os.File, str string) (string, error) {

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()

		//line == "" handles both cases transparently because bufio.Scanner automatically strips \r\n(Windows) or \n(Linux/macOS)
		if line == "" {
			break
		} else if strings.HasPrefix(strings.ToLower(line), strings.ToLower(str)) {
			return strings.TrimSpace(line[len("from:"):]), nil // "null"
		}
	}

	return "", fmt.Errorf("\"From\" header missing or value is empty")
}

func readTestStrings(filename string) ([]string, error) {

	lines := []string{}

	fd, err := os.Open(filename)
	if err != nil {
		fmt.Printf(err.Error())
		return nil, err
	}

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()

		//line == "" handles both cases transparently because bufio.Scanner automatically strips \r\n(Windows) or \n(Linux/macOS)
		if line == "" {
			break
		}
		lines = append(lines, line)
	}

	//close the opened file and check for error
	err = fd.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return lines, nil
}

// run test cases using different combinations of display name and email address
//
// Return void
func doCustomFileTests(filename string) {
	var emails, err = readTestStrings(filename)

	if err != nil {
		fmt.Println(err)
	}

	// test each input string in the array
	for _, fromStr := range emails {
		info := make(map[string]string)

		fromStr, err := checkForErrors(fromStr)
		if err != nil {
			info["display_name"] = ""
			info["addr_spec"] = ""
			info["error"] = err.Error()
		} else {
			info, err = extractEmailInfo(fromStr)
			info["display_name"] = strings.ReplaceAll(info["display_name"], "<", "")
			info["addr_spec"] = strings.ReplaceAll(info["addr_spec"], ">", "")
		}
		displayData(info, err)
	}
}

// Extracts the display name and email address from a string
func extractEmailInfo(input string) (map[string]string, error) {

	retVal := make(map[string]string)

	//First, clean the input by trimming whitespace and special chars
	input = strings.ReplaceAll(input, "“", `"`)
	input = strings.ReplaceAll(input, "”", `"`)
	input = strings.ReplaceAll(input, "\"", ``)

	//workhorse of the application
	//parses the input string extracted from the email
	retVal = parseDisplayNameAndEmail(input)
	return retVal, nil
}

// remove nested comments in a string if they exits
//
// Returns a string without comments in "()"
func removeNestedComments(s string) string {

	var sb strings.Builder
	depth := 0

	for i := 0; i < len(s); i++ {
		char := s[i]

		if char == '(' {
			depth++
		}

		if depth == 0 {
			sb.WriteByte(char)
		}

		if char == ')' && depth > 0 {
			depth--
		}
	}

	return strings.TrimSpace(sb.String())
}

// check the "From:" string for validation
// Also makes some small transformation of the input string
// validates the from line against :
// 1. nested <> in addr_spec
// 2. missing @ domain
// 3. no addr-spec found
// 4. RFC 5322 forbids the localpart (what comes before the last @ in addr-spec) from ending in a dot
// 5. more than one addr-spec given
// 6. unterminated quoted part
//
// Returns an error if the validation does not passes
// or input string with small transformation for following analysis in detection
func checkForErrors(str string) (string, error) {

	str = strings.Trim(str, "\n\r")

	brackets := strings.Contains(str, ">>") || strings.Contains(str, "<<")

	if brackets {
		return "", fmt.Errorf("nested < .. > not allowed as part of addr-spec")
	}

	checkEmailSym := strings.Split(str, "@")
	if strings.Contains(str, "<") && len(checkEmailSym) == 1 {
		return "", fmt.Errorf("missing @ domain")
	}

	if len(checkEmailSym) == 1 {
		return "", fmt.Errorf("no addr-spec found")
	}

	userName, domain := checkEmailSym[0], checkEmailSym[1]
	if strings.HasPrefix(userName, ".") || strings.HasSuffix(userName, ".") || strings.HasPrefix(domain, ".") {
		return "", fmt.Errorf("RFC 5322 forbids the localpart (what comes before the last @ in addr-spec) from ending in a dot")
	}

	emailSplit := strings.Split(str, "\"")
	if len(emailSplit) == 3 {
		numEmails := countNoEmails(emailSplit[2])
		if numEmails > 1 {
			fmt.Printf("")
			return "", fmt.Errorf("more than one addr-spec given")
		}
	}

	if len(emailSplit) > 1 && strings.Contains(emailSplit[1], "<") && strings.Contains(emailSplit[1], ">") {
		str = strings.Replace(str, "<", "(", 1)
		str = strings.Replace(str, ">", ")", 1)
		str = removeNestedComments(str)
	}
	noQuotes := 0
	noQuotes = strings.Count(str, "\"")

	noEscQuotes := strings.Count(str, "\\\"")
	if noEscQuotes > 0 {
		if noQuotes%2 == 0 {
			str = strings.Replace(str, "\\\"", "", noEscQuotes)
		}
		noEscQuotes = noEscQuotes - 1
		noQuotes = noQuotes - 1
	}

	if noEscQuotes%2 != 0 || noQuotes%2 != 0 {
		return "", fmt.Errorf("unterminated quoted part")
	}

	return str, nil
}

// count how many email are in a "from:" string
//
// Return number of email in the pattern name@web.com with and without <> ()
func countNoEmails(input string) int {
	// Regex matches content within < > that contains an @ symbol
	re := regexp.MustCompile(`<([^>]+@[^>]+)>`)
	matches := re.FindAllStringSubmatch(input, -1)

	return len(matches)
}

// extract the display name and email address from a string
// using 4 different representations of a display name and email
//
// Returns a map of a (display name and an email)
func parseDisplayNameAndEmail(str string) map[string]string {
	retVal := make(map[string]string)

	str = removeNestedComments(str)
	str = strings.TrimSpace(str)

	// 1st try: display name and <email>
	bracketRe := regexp.MustCompile(`(?i)^"?([^"<]*)"?\s*<\s*([^@\s<>]+@[^@\s<>]+\.[^@\s<>]+)\s*>$`)
	if m := bracketRe.FindStringSubmatch(str); m != nil {
		retVal["display_name"] = m[1]
		retVal["addr_spec"] = m[2]

		return retVal
	}

	// 2nd try: display name and bare email(no angle brackets)
	bareNameEmailRe := regexp.MustCompile(`(?i)^([^<"\s@][^<@"]*)\s+([a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})$`)
	if m := bareNameEmailRe.FindStringSubmatch(str); m != nil {
		retVal["display_name"] = m[1]
		retVal["addr_spec"] = m[2]

		return retVal
	}

	// 3rd try: just angle brackets email
	bracketOnlyRe := regexp.MustCompile(`(?i)^<\s*([^@\s<>]+@[^@\s<>]+\.[^@\s<>]+)\s*>$`)
	if m := bracketOnlyRe.FindStringSubmatch(str); m != nil {
		retVal["display_name"] = ""
		retVal["addr_spec"] = m[1]

		return retVal
	}

	// 4th try: just plain email only, no angle brackets
	emailRe := regexp.MustCompile(`(?i)^([a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})$`)
	if m := emailRe.FindStringSubmatch(str); m != nil {
		retVal["display_name"] = ""
		retVal["addr_spec"] = m[1]

		return retVal
	}
	return retVal
}

// build a json structure from a structure
//
// Returns a json byte array
func createJSONOutput(output jsonOutput) []byte {

	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error generating JSON output: %v", err)
		os.Exit(1)
	}
	return jsonOutput
}
