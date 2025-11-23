package wirefilter_test

import (
	"fmt"
	"log"

	"github.com/vitalvas/gokit/wirefilter"
)

func ExampleCompile() {
	schema := wirefilter.NewSchema().
		AddField("http.host", wirefilter.TypeString).
		AddField("http.status", wirefilter.TypeInt)

	filter, err := wirefilter.Compile(`http.host == "example.com" and http.status >= 400`, schema)
	if err != nil {
		log.Fatal(err)
	}

	ctx := wirefilter.NewExecutionContext().
		SetStringField("http.host", "example.com").
		SetIntField("http.status", 500)

	result, err := filter.Execute(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// Output: true
}

func ExampleFilter_Execute() {
	schema := wirefilter.NewSchema().
		AddField("http.method", wirefilter.TypeString).
		AddField("http.path", wirefilter.TypeString)

	filter, err := wirefilter.Compile(`http.method == "GET" and http.path contains "/api"`, schema)
	if err != nil {
		log.Fatal(err)
	}

	ctx := wirefilter.NewExecutionContext().
		SetStringField("http.method", "GET").
		SetStringField("http.path", "/api/v1/users")

	result, err := filter.Execute(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// Output: true
}

func ExampleNewSchema() {
	schema := wirefilter.NewSchema().
		AddField("http.host", wirefilter.TypeString).
		AddField("http.status", wirefilter.TypeInt).
		AddField("http.secure", wirefilter.TypeBool).
		AddField("ip.src", wirefilter.TypeIP)

	filter, err := wirefilter.Compile(`http.status >= 200 and http.status < 300`, schema)
	if err != nil {
		log.Fatal(err)
	}

	ctx := wirefilter.NewExecutionContext().
		SetIntField("http.status", 200)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func ExampleNewExecutionContext() {
	ctx := wirefilter.NewExecutionContext().
		SetStringField("http.host", "example.com").
		SetIntField("http.status", 200).
		SetBoolField("http.secure", true).
		SetIPField("ip.src", "192.168.1.1")

	schema := wirefilter.NewSchema().
		AddField("http.host", wirefilter.TypeString).
		AddField("http.status", wirefilter.TypeInt).
		AddField("http.secure", wirefilter.TypeBool).
		AddField("ip.src", wirefilter.TypeIP)

	filter, _ := wirefilter.Compile(`http.host == "example.com" and http.secure == true`, schema)
	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_stringOperations() {
	schema := wirefilter.NewSchema().
		AddField("http.path", wirefilter.TypeString).
		AddField("http.user_agent", wirefilter.TypeString)

	filter, _ := wirefilter.Compile(`http.path contains "/api" and http.user_agent matches "^Mozilla.*"`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetStringField("http.path", "/api/v1/users").
		SetStringField("http.user_agent", "Mozilla/5.0")

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_ipFiltering() {
	schema := wirefilter.NewSchema().
		AddField("ip.src", wirefilter.TypeIP).
		AddField("ip.dst", wirefilter.TypeIP)

	filter, _ := wirefilter.Compile(`ip.src in "192.168.0.0/16"`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetIPField("ip.src", "192.168.1.100")

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_ipv6Filtering() {
	schema := wirefilter.NewSchema().
		AddField("ip.src", wirefilter.TypeIP)

	filter, _ := wirefilter.Compile(`ip.src in "2001:db8::/32"`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetIPField("ip.src", "2001:db8::1")

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_arrayMembership() {
	schema := wirefilter.NewSchema().
		AddField("http.status", wirefilter.TypeInt)

	filter, _ := wirefilter.Compile(`http.status in {200, 201, 204}`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetIntField("http.status", 200)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_rangeExpression() {
	schema := wirefilter.NewSchema().
		AddField("http.status", wirefilter.TypeInt)

	filter, _ := wirefilter.Compile(`http.status in {200..299}`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetIntField("http.status", 250)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_multipleRanges() {
	schema := wirefilter.NewSchema().
		AddField("port", wirefilter.TypeInt)

	filter, _ := wirefilter.Compile(`port in {80..100, 443, 8000..9000}`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetIntField("port", 443)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_booleanLogic() {
	schema := wirefilter.NewSchema().
		AddField("http.secure", wirefilter.TypeBool).
		AddField("http.status", wirefilter.TypeInt)

	filter, _ := wirefilter.Compile(`http.secure == true and http.status == 200`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetBoolField("http.secure", true).
		SetIntField("http.status", 200)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_notOperator() {
	schema := wirefilter.NewSchema().
		AddField("http.status", wirefilter.TypeInt)

	filter, _ := wirefilter.Compile(`not (http.status >= 500)`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetIntField("http.status", 200)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_orOperator() {
	schema := wirefilter.NewSchema().
		AddField("http.status", wirefilter.TypeInt)

	filter, _ := wirefilter.Compile(`http.status == 404 or http.status == 500`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetIntField("http.status", 404)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_complexExpression() {
	schema := wirefilter.NewSchema().
		AddField("http.host", wirefilter.TypeString).
		AddField("http.method", wirefilter.TypeString).
		AddField("http.status", wirefilter.TypeInt).
		AddField("http.path", wirefilter.TypeString)

	expression := `
		(http.host == "api.example.com" or http.host == "api.test.com") and
		http.method == "GET" and
		http.path contains "/v1/" and
		http.status >= 200 and http.status < 300
	`

	filter, _ := wirefilter.Compile(expression, schema)

	ctx := wirefilter.NewExecutionContext().
		SetStringField("http.host", "api.example.com").
		SetStringField("http.method", "GET").
		SetStringField("http.path", "/v1/users").
		SetIntField("http.status", 200)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_networkTrafficFiltering() {
	schema := wirefilter.NewSchema().
		AddField("ip.src", wirefilter.TypeIP).
		AddField("port.dst", wirefilter.TypeInt).
		AddField("protocol", wirefilter.TypeString)

	expression := `
		ip.src in "10.0.0.0/8" and
		port.dst in {80, 443, 8080..8090} and
		protocol == "tcp"
	`

	filter, _ := wirefilter.Compile(expression, schema)

	ctx := wirefilter.NewExecutionContext().
		SetIPField("ip.src", "10.1.2.3").
		SetIntField("port.dst", 443).
		SetStringField("protocol", "tcp")

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_allEqualOperator() {
	schema := wirefilter.NewSchema().
		AddField("tags", wirefilter.TypeArray)

	filter, _ := wirefilter.Compile(`tags === "production"`, schema)

	tags := wirefilter.ArrayValue{
		wirefilter.StringValue("production"),
		wirefilter.StringValue("production"),
		wirefilter.StringValue("production"),
	}

	ctx := wirefilter.NewExecutionContext().
		SetField("tags", tags)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_anyNotEqualOperator() {
	schema := wirefilter.NewSchema().
		AddField("tags", wirefilter.TypeArray)

	filter, _ := wirefilter.Compile(`tags !== "deprecated"`, schema)

	tags := wirefilter.ArrayValue{
		wirefilter.StringValue("production"),
		wirefilter.StringValue("critical"),
		wirefilter.StringValue("deprecated"),
	}

	ctx := wirefilter.NewExecutionContext().
		SetField("tags", tags)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_noSchemaValidation() {
	filter, err := wirefilter.Compile(`http.host == "example.com" and http.status >= 400`, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := wirefilter.NewExecutionContext().
		SetStringField("http.host", "example.com").
		SetIntField("http.status", 500)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_schemaWithFieldsMap() {
	fields := map[string]wirefilter.Type{
		"http.host":   wirefilter.TypeString,
		"http.status": wirefilter.TypeInt,
		"http.secure": wirefilter.TypeBool,
	}

	schema := wirefilter.NewSchema(fields)

	filter, _ := wirefilter.Compile(`http.host == "example.com" and http.status == 200 and http.secure == true`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetStringField("http.host", "example.com").
		SetIntField("http.status", 200).
		SetBoolField("http.secure", true)

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}

func Example_schemaWithMultipleFieldMaps() {
	httpFields := map[string]wirefilter.Type{
		"http.host":   wirefilter.TypeString,
		"http.method": wirefilter.TypeString,
		"http.status": wirefilter.TypeInt,
	}

	networkFields := map[string]wirefilter.Type{
		"ip.src": wirefilter.TypeIP,
		"ip.dst": wirefilter.TypeIP,
	}

	schema := wirefilter.NewSchema(httpFields, networkFields)

	filter, _ := wirefilter.Compile(`http.method == "GET" and ip.src in "10.0.0.0/8"`, schema)

	ctx := wirefilter.NewExecutionContext().
		SetStringField("http.method", "GET").
		SetIPField("ip.src", "10.1.2.3")

	result, _ := filter.Execute(ctx)
	fmt.Println(result)
	// Output: true
}
