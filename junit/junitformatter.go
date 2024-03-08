package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/hive/internal/libhive"
)

// TestSuites represents the aggregated test results.
type TestSuites struct {
	XMLName    xml.Name `xml:"testsuites"`
	Failures   int      `xml:"failures,attr"`
	Name       string   `xml:"name,attr"`
	Tests      int      `xml:"tests,attr"`
	Suites     []TestSuite
}

// TestSuite represents a single test suite.
type TestSuite struct {
	XMLName       xml.Name `xml:"testsuite"`
	Name          string   `xml:"name,attr"`
	Failures      int      `xml:"failures,attr"`
	Tests         int      `xml:"tests,attr"`
	Properties    Properties
	TestCases     []TestCase
}

// Properties holds various properties of a test suite.
type Properties struct {
	Properties []Property
}

// Property represents a single property.
type Property struct {
	Name string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// TestCase represents a single test case.
type TestCase struct {
	XMLName xml.Name `xml:"testcase"`
	Name     string   `xml:"name,attr"`
	Time     string   `xml:"time,attr"`
	Failure *Failure
	SystemOut string `xml:"system-out"`
}

// Failure represents a test failure.
type Failure struct {
	Message string `xml:"message,attr"`
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("Error: no input files specified")
		os.Exit(1)
	}

	result := TestSuites{
		Name: "Hive Results",
	}
	var suites []TestSuite

	for _, file := range os.Args[1:] {
		suite, err := readInput(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		junitSuite := mapTestSuite(suite)
		result.Failures += junitSuite.Failures
		result.Tests += junitSuite.Tests
		suites = append(suites, junitSuite)
	}
	result.Suites = suites

	junit, err := xml.MarshalIndent(result, "", " ")
	if err != nil {
		fmt.Println("Error marshalling XML:", err)
		os.Exit(1)
	}
	fmt.Println(string(junit))
}

func readInput(file string) (libhive.TestSuite, error) {
	inData, err := os.ReadFile(file)
	if err != nil {
		return libhive.TestSuite{}, fmt.Errorf("failed to read file '%v': %w", file, err)
	}

	var suite libhive.TestSuite
	err = json.Unmarshal(inData, &suite)
	if err != nil {
		return libhive.TestSuite{}, fmt.Errorf("failed to parse file '%v': %w", file, err)
	}
	return suite, nil
}

func mapTestSuite(suite libhive.TestSuite) TestSuite {
	junitSuite := TestSuite{
		Name:       suite.Name,
		Tests:      len(suite.TestCases),
		Properties: Properties{},
	}
	for clientName, clientVersion := range suite.ClientVersions {
		junitSuite.Properties.Properties = append(junitSuite.Properties.Properties, Property{
			Name: clientName,
			Value: clientVersion,
		})
	}
	for _, testCase := range suite.TestCases {
		if !testCase.SummaryResult.Pass {
			junitSuite.Failures++
		}
		junitSuite.TestCases = append(junitSuite.TestCases, mapTestCase(testCase))
	}
	return junitSuite
}

func mapTestCase(source *libhive.TestCase) TestCase {
	result := TestCase{
		Name: source.Name,
	}
	if source.SummaryResult.Pass {
		result.SystemOut = source.SummaryResult.Details
	} else {
		result.Failure = &Failure{Message: source.SummaryResult.Details}
	}
	duration := source.End.Sub(source.Start)
	result.Time = strconv.FormatFloat(duration.Seconds(), 'f', 6, 64)
	return result
}


/*
Target XML format (lots of it being optional):
<testsuites disabled="" errors="" failures="" name="" tests="" time="">
    <testsuite disabled="" errors="" failures="" hostname="" id=""
               name="" package="" skipped="" tests="" time="" timestamp="">
        <properties>
            <property name="" value=""/>
        </properties>
        <testcase assertions="" classname="" name="" status="" time="">
            <skipped/>
            <error message="" type=""/>
            <failure message="" type=""/>
            <system-out/>
            <system-err/>
        </testcase>
        <system-out/>
        <system-err/>
    </testsuite>
</testsuites>
*/

type TestSuites struct {
	XMLName  string      `xml:"testsuites,omitempty"`
	Failures int         `xml:"failures,attr"`
	Name     string      `xml:"name,attr"`
	Tests    int         `xml:"tests,attr"`
	Suites   []TestSuite `xml:"testsuite"`
}

type TestSuite struct {
	Name       string     `xml:"name,attr"`
	Failures   int        `xml:"failures,attr"`
	Tests      int        `xml:"tests,attr"`
	Properties Properties `xml:"properties,omitempty"`
	TestCases  []TestCase `xml:"testcase"`
}

type TestCase struct {
	Name      string   `xml:"name,attr"`
	Time      string   `xml:"time,attr"`
	Failure   *Failure `xml:"failure,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
}

type Failure struct {
	Message string `xml:"message,attr"`
}

type Properties struct {
	Properties []Property `xml:"property,omitempty"`
}

type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}
