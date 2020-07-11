package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
	// The following package provides simplistic plain text diff representation
	// With diff lines designated with - and +
	// This script uses it for writing diffs to files
	df "github.com/kylelemons/godebug/diff"
)

const (
	MainNetURL = "https://wallet-api.insolar.io/api"
	//TestNetURL = "https://wallet-api.testnet.insolar.io/api"
	//LocalURL = "http://127.0.0.1:8080/api/"
	Endpoint = "/transactions/inPulseNumberRange"
)

func sendRequest(URLandEndpoint string, params url.Values) (string, string) {

	// Initialize the client
	client := http.Client{}

	// Create a new GET request
	transactionsReq, err := http.NewRequest("GET", URLandEndpoint, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Add query params
	transactionsReq.URL.RawQuery = params.Encode()
	fmt.Println("GET " + transactionsReq.URL.String())

	// Send the request
	transactionsResp, err := client.Do(transactionsReq)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer transactionsResp.Body.Close()

	// Read the response body
	transactionsRespBody, _ := ioutil.ReadAll(transactionsResp.Body)
	fmt.Println("Response status: " + transactionsResp.Status)

	// Unmarshal the response into a JSON
	var rawBody []interface{}
	err = json.Unmarshal(transactionsRespBody, &rawBody)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Prettify the JSON: add indentation for better human-readability
	prettyRespJSON, err := json.MarshalIndent(rawBody, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Return both the response as it is returned by the API (a one-line string)
	// and the beautiful JSON for pretty-printing
	return string(transactionsRespBody), string(prettyRespJSON)
}

func main() {
	// To provide flexibility, we parametrize the script with:
	// - fromPulseNumber and toPulseNumber to specify any pulse number range
	// - limit to the number of returned transactions (just common sense)
	// - local API endpoint because we can't predict how the API is going to be configured on the client side

	if len(os.Args) != 5 {
		fmt.Println("Usage:\n  go run pulse_comparator.go <fromPulseNumber> <toPulseNumber> <limit> <http[s]://your-node-s-api-endpoint>[:port]/api>")
		fmt.Println("\n  — <fromPulseNumber> value must chronologically precede <toPulseNumber> or they may be the same.")
		fmt.Println("  — <limit> can take values from 1 to 1000.")
		fmt.Println("  — The last parameter is your node's API endpoint as you configured it.")
		os.Exit(1)
	}

	params:= url.Values{}
	params.Add("fromPulseNumber", os.Args[1])
	params.Add("toPulseNumber", os.Args[2])
	params.Add("limit", os.Args[3])

	// Save the request to a file to be able to copy-paste and send the exact same request later
	// To do so, first, create a file and put the current date and time into its name
	currentTime := time.Now().Format(time.UnixDate)
	file, err := os.Create( currentTime + ".txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	// Then write the request URL with all query parameters to it
	requestString := MainNetURL + Endpoint + "?" + params.Encode()
	file.WriteString("The request is as follows:\n\n")
	file.WriteString(requestString + "\n\n")

	// Send the request to MainNet using the sender function declared earlier and receive pulse contents
	fmt.Println("\nGetting transactions within a pulse number range from MainNet with the following request:")
	mainNetPulseContent, prettyMainNetPulseContent := sendRequest(MainNetURL + Endpoint, params)

	// To test the case with a difference, comment function that sends request to the node
	// And uncomment the following lines

	//nodePulseContent := `[{"amount":"4045300000000","fee":"100000000","index":"48222814:81","pulseNumber":48222814,"status":"received","timestamp":1594458077,"txID":"insolar:1At_SXhIyDsmIM0z4bRRoa8vu1-xPmzOmJGzDr029yJo.record","fromMemberReference":"insolar:1At9mtqXJbRnsPSLf4Lmb-G7M92ChDmtERC64li7JI8Q","toMemberReference":"insolar:1Ag6rgI9GWg8ODZ0Mf3At8ehc1HYQUFbivJFkCVGn59s","type":"transfer"}]`
	//var rawBody []interface{}
	//err = json.Unmarshal([]byte(mainNetPulseContent), &rawBody)
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//
	//indentedTransactions1Byte, err := json.MarshalIndent(rawBody, "", "  ")
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//prettyNodePulseContent := string(indentedTransactions1Byte)

	// Send the exact same request to your node and receive pulse contents
	fmt.Println("\nGetting transactions within a pulse number range locally with the following request:")
	nodePulseContent, prettyNodePulseContent := sendRequest(MainNetURL + Endpoint, params)

	// Compare the contents with basic Golang comparison operator (==)
	if mainNetPulseContent == nodePulseContent {
		// This is the case with identical pulse contents from both MainNet and your node

		// Write the pulse contents to the file to be able to later compare it to the results of the exact same request
		file.WriteString("Pulse contents from MainNet and your node are identical. Here's the response:\n\n")
		file.WriteString(mainNetPulseContent)
		file.Close()
		fmt.Println("\nThe request and result it returned are saved to the \"" + currentTime + ".txt\" file.")

	} else {
		// This is the case with pulse contents that differ
		// Remember: some transaction attributes may change their values since the values depend on separate objects
		// But each pulse always contains the same set of registered transactions

		// Form a simple plaint text diff between prettified reponse—JSON object with indentation
		indentedDiff := df.Diff(prettyMainNetPulseContent, prettyNodePulseContent)

		// Write the diff to the file
		file.WriteString("The diff goes as follows:\n\n")
		file.WriteString(indentedDiff)
		file.Close()
		fmt.Println("\nThe request and diff are saved to the \"" + currentTime + ".txt\" file.")
	}
}
