package main

import (
	"encoding/json"
	"fmt"
	// Provides simplistic plain text diff representation with diff lines designated with - and +.
	// This script uses it for writing diffs to files.
	df "github.com/kylelemons/godebug/diff"

	// Provides pretty diff representation for terminal with diff lines designated with red and green colors.
	dff "github.com/sergi/go-diff/diffmatchpatch"

	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	MainNetURL = "https://wallet-api.insolar.io/api"
	TestNetURL = "https://wallet-api.testnet.insolar.io/api"
	//LocalURL = "http://127.0.0.1:8080/api/"
	Endpoint = "/transactions/inPulseNumberRange"
)

func sendRequest(URL string, params url.Values) (string, string) {

	// Initialize the client
	client := http.Client{}

	// Create a new GET request
	transactionsReq, err := http.NewRequest("GET", URL, nil)
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

	// Prettify the JSON
	prettyRespJSON, err := json.MarshalIndent(rawBody, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Return both the response as it is returned by the API (a one-line string) and the beautiful JSON for pretty-printing
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
	requestString := MainNetURL + Endpoint + "?" + params.Encode()

	fmt.Println("\nGetting transactions within a pulse number range from MainNet with the following request:\n")

	mainPulseCntnt, prettyMainPulseCntnt := sendRequest(MainNetURL + Endpoint, params)

	//Comment the previous line and uncomment the following to test the case with a difference.
	//mainPulseCntnt := `[{"amount":"100000000000000","fee":"","index":"47289127:51","pulseNumber":47289127,"status":"some_different_status","timestamp":1593524390,"txID":"insolar:1AtGTJ8ytEnaESprhR3VOd1UtSK0S2t7YeLjnTht10AM.record","fromMemberReference":"insolar:1AAEAAWeNhA_NwKaH6E36IJ-2PLvXnJRxiTTNWq1giOg","toMemberReference":"insolar:1AtGTJwUKe7my117vMZPBBpzTEam-wJx6aWxeAtE4ZKw","type":"transfer"},{"amount":"100000000000000","fee":"","index":"47289127:111","pulseNumber":47289127,"status":"failed","timestamp":1593524390,"txID":"insolar:1AtGTJzrPe8qXiEJ2Gin4_YEPK05ttVG4QrnlH27qLbE.record","fromMemberReference":"insolar:1AAEAAWeNhA_NwKaH6E36IJ-2PLvXnJRxiTTNWq1giOg","toMemberReference":"insolar:1AtGTJx_F-66HR4LGxckXlBmNldI2KA9cGYKtXngsJVs","type":"transfer"},{"amount":"100","fee":"0","index":"47289127:376","pulseNumber":47289127,"status":"received","timestamp":1593524390,"txID":"insolar:1AtGTJ_RiMvX27EWE6ROOleCQ329YAdrFcXuP1ZTkzmA.record","fromMemberReference":"insolar:1AAEAAS2JLnvh2bkLUUlUijqcp7--k8_GIz9qLKxZXLE","toDepositReference":"insolar:1AtGTJ490s4HHcgfO9vl-o-kooZVTyJFlaquRgQPx6EI","toMemberReference":"insolar:1AtGTJ2Hb3j_58uR2iPpHu9p4_WOJpyGVfaew9uDhQrU","type":"migration"}]`
	//var rawBody []interface{}
	//err := json.Unmarshal([]byte(mainPulseCntnt), &rawBody)
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
	//prettyMainPulseCntnt := string(indentedTransactions1Byte)

	fmt.Println("\nGetting transactions within a pulse number range locally with the following request:\n")
	nodePulseCntnt, prettyNodePulseCntnt := sendRequest(os.Args[4] + Endpoint, params)

	// Comparing strings with basic Golang comparison operator (==) as it's the most reliable.
	if mainPulseCntnt == nodePulseCntnt {

		// This prints out the pretty JSON to the terminal.
		fmt.Println("\nNow comparing:\n\n"+ prettyMainPulseCntnt)

		print := "\nPulse contents from both sources are identical." + "\n\n" +
			"In case you want to make sure that the returned array did not change over time, here's the request to send later:" + "\n\n" +
			requestString + "\n"
		fmt.Println(print)

		currentTime := time.Now().Format(time.UnixDate)
		file, err := os.Create("Request and result received at " + currentTime + ".txt")
		if err != nil {
			fmt.Println(err)
			return
		}
		file.WriteString("Open the following URL in the browser:" + "\n\n")
		file.WriteString(requestString + "\n\n")
		file.WriteString("Copy-paste both the output below and the result in the browser and compare them, for example, using an online diff tool." + "\n\n")
		// This writes a one-liner request to the file for future copy-paste.
		// This one-liner is supposed to be compared with a new request output in the browser which is a one-liner as well.
		file.WriteString(mainPulseCntnt)
		file.Close()
		fmt.Println("Just in case: the request and array it returned are saved to the \"Request and array received at "+ currentTime + ".txt\" file.")

	} else {

		// Forming a human-readable diff for the terminal.

		dmpDiff2 := dff.New()
		indentedDiff2 := dmpDiff2.DiffMain(prettyMainPulseCntnt, prettyNodePulseCntnt, true)
		prettyIndentedDiff2 := dmpDiff2.DiffPrettyText(indentedDiff2)

		fmt.Println("\nNow comparing:\n\n" + prettyIndentedDiff2)
		print := "\nThere's a difference!" + "\n\n" +
			"Remember: transaction statuses and fees (and, in some cases, fromDepositReference of migration transactions) may change as transactions go through the finalization process." + "\n" +
			"This may result in differences in corresponding transaction attributes between MainNet and your node." + "\n" +
			"Tip: wait for transactions to finalize and your node to sync with MainNet and repeat the request or, simply, pick an \"older\" pulse." + "\n\n"
		fmt.Println(print)

		// Saving plain text diff to file.
		indentedDiff1 := df.Diff(prettyMainPulseCntnt, prettyNodePulseCntnt)
		currentTime := time.Now().Format(time.Stamp)
		file, err := os.Create("Request and diff received at " + currentTime + ".txt")
		if err != nil {
			fmt.Println(err)
			return
		}
		file.WriteString("The request goes as follows:\n\n" + requestString + "\n\n")
		file.WriteString(print)
		file.WriteString(indentedDiff1)
		file.Close()
		fmt.Println("Just in case: the request and diff are saved to the \"Request and diff received at "+ currentTime + ".txt\" file.")
	}
}
