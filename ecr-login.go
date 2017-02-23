package main

import (
	"encoding/base64"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

// Auth structure
type Auth struct {
	Token         string
	User          string
	Pass          string
	ProxyEndpoint string
	ExpiresAt     time.Time
}

// error handler
func check(e error) {
	if e != nil {
		panic(e.Error())
	}
}

// DefaultTemplate prints docker login command
const DefaultTemplate = `{{range .}}docker login -u {{.User}} -p {{.Pass}} {{.ProxyEndpoint}}
{{end}}`

// load template from file or use default
func getTemplate() *template.Template {
	var tmpl *template.Template
	var err error

	file, exists := os.LookupEnv("TEMPLATE")

	if exists {
		tmpl, err = template.ParseFiles(file)
	} else {
		tmpl, err = template.New("default").Parse(DefaultTemplate)
	}

	check(err)
	return tmpl
}

// if AWS_REGION not set, infer from instance metadata
func getRegion(sess *session.Session) string {
	region, exists := os.LookupEnv("AWS_REGION")
	if !exists {
		ec2region, err := ec2metadata.New(sess).Region()
		check(err)
		region = ec2region
	}
	return region
}

// get list of registries from env, leave empty for default
func getRegistryIds() []*string {
	var registryIds []*string
	registries, exists := os.LookupEnv("REGISTRIES")
	if exists {
		for _, registry := range strings.Split(registries, ",") {
			registryIds = append(registryIds, aws.String(registry))
		}
	}
	return registryIds
}

func main() {
	// configure aws client
	sess := session.New()
	svc := ecr.New(sess, aws.NewConfig().WithRegion(getRegion(sess)))

	// this lets us handle multiple registries
	params := &ecr.GetAuthorizationTokenInput{
		RegistryIds: getRegistryIds(),
	}

	// request the token
	resp, err := svc.GetAuthorizationToken(params)
	check(err)

	// fields to send to template
	fields := make([]Auth, len(resp.AuthorizationData))
	for i, auth := range resp.AuthorizationData {

		// extract base64 token
		data, err := base64.StdEncoding.DecodeString(*auth.AuthorizationToken)
		check(err)

		// extract username and password
		token := strings.SplitN(string(data), ":", 2)

		// object to pass to template
		fields[i] = Auth{
			Token:         *auth.AuthorizationToken,
			User:          token[0],
			Pass:          token[1],
			ProxyEndpoint: *(auth.ProxyEndpoint),
			ExpiresAt:     *(auth.ExpiresAt),
		}
	}

	// run the template
	err = getTemplate().Execute(os.Stdout, fields)
	check(err)
}
