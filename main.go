package main

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"
)


func main() {

	e := echo.New()

	//e.Use(middlewareOne)
	//e.Use(middlewareTwo)
	//e.Use(echo.WrapMiddleware(middlewareSomething))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method | uri=${uri} | status=${status} | error=${error} | protokol=${protocol}" +
			"host=${host} | path=${path} | referer=${referer} | user_agent=${user_agent}\n",
	}))

	e.Use(middlewareLogging)
	e.HTTPErrorHandler = errorHandler

	e.Static("/static", "assets")

	e.GET("/", func(context echo.Context) error {
		data := "FROM ROOT"
		return context.String(http.StatusOK, data)
	})
	e.GET("/index", func(context echo.Context) error {
		return context.Redirect(http.StatusTemporaryRedirect, "/")
	})
	e.GET("/articles", func(context echo.Context) error {
		data := M{1: "Learn Java", 2: "Learn Golang", 3: "Learn Microservice",}
		return context.JSON(http.StatusOK, data)
	})

	// ===> curl -X GET http://localhost:9000/page1?name=sammidev
	e.GET("/page1", func(context echo.Context) error {
		name := context.QueryParam("name")
		data := fmt.Sprintf("hello %s", name)
		return context.String(http.StatusOK, data)
	})

	// ===> curl -X GET http://localhost:9000/page2/sammidev
	e.GET("/page2/:name", func(context echo.Context) error {
		name := context.Param("name")
		data := fmt.Sprintf("hello %s", name)
		return context.String(http.StatusOK, data)
	})

	// ===> curl -X GET http://localhost:9000/page3/tim/werlcome/blablabla
	e.GET("/page3/:name/*", func(ctx echo.Context) error {
		name := ctx.Param("name")
		message := ctx.Param("*")
		data := fmt.Sprintf("Hello %s, I have message for you: %s", name, message)
		return ctx.String(http.StatusOK, data)
	})

	// ===> curl -X POST -F name=sammidev -F dream=wellcome http://localhost:9000/page4
	e.POST("/page4", func(context echo.Context) error {
		name := context.FormValue("name")
		dream := context.FormValue("dream")
		data := fmt.Sprintf(
			"Hello %s, I have message for you: %s",
			name,
			strings.Replace(dream, "/", "", 1),
		)
		return context.String(http.StatusOK, data)

	})
	e.GET("/about", echo.WrapHandler(About))


	// parsing payload
	// curl -X POST http://localhost:9000/student -H 'Content-Type: application/json' -d '{"nisn":"12435", "name":"Sammidev", "age":19}'
	// curl -X POST http://localhost:9000/student -H 'Content-Type: application/xml' -d '<?xml version="1.0"?><Data><Nisn>12345</Nisn><Name>Sam</Name><Age>19</Age></Data>'
	// curl -X POST http://localhost:9000/student -d 'nisn=12345' -d 'name=sam' -d 'age=19'
	// curl -X GET http://localhost:9000/student?nisn=12345&name=Sam&age=19

	e.Any("/student", func(context echo.Context) (err error) {
		u := new(Student)
		if err = context.Bind(u); err != nil {
			return
		}
		return context.JSON(http.StatusOK, u)
	})

	// validation
	e.Validator = &CustomValidator{validator: validator.New()}
	e.POST("/employee", func(context echo.Context) error {
		u := new(Employee)
		if err := context.Bind(u); err != nil {
			return err
		}
		if err := context.Validate(u); err != nil {
			return err
		}
		return context.JSON(http.StatusOK, true)
	})

	// template rendering
	e.Renderer = NewRenderer("./*.html", true)
	e.GET("/testest", func(context echo.Context) error {
		data := A{"message": "Hello World!"}
		return context.Render(http.StatusOK, "index.html", data)
	})
	lock := make(chan error)
	go func(lock chan error) {
		lock <- e.Start(":9000")
	}(lock)
	time.Sleep(1 * time.Millisecond)
	makeLogEntry(nil).Warning(" STARTING WITHOUT SSL/TLS ENALBLED")
	err := <-lock
	if err != nil {
		makeLogEntry(nil).Panic("FAILED TO START APP")
	}
}

type A map[string]interface{}
type M map[int]interface{}

type Renderer struct {
	template *template.Template
	debug bool
	location string
}
func NewRenderer(location string, debug bool) *Renderer {
	tpl := new(Renderer)
	tpl.location = location
	tpl.debug = debug

	tpl.ReloadTemplates()

	return tpl
}

type CustomValidator struct {
	validator *validator.Validate
}
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
func (t *Renderer) ReloadTemplates() {
	t.template = template.Must(template.ParseGlob(t.location))
}
func (t *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	if t.debug {
		t.ReloadTemplates()
	}

	return t.template.ExecuteTemplate(w, name, data)
}

func makeLogEntry(c echo.Context) *log.Entry {
	if c == nil {
		return log.WithFields(log.Fields{
			"at": time.Now().Format("2020-09-01 15:04:05"),
		})
	}
	return log.WithFields(log.Fields{
		"at ": time.Now().Format("2020-09-01 15:04:05"),
		"method ": c.Request().Method,
		"uri ": c.Request().URL.String(),
		"ip ": c.Request().RemoteAddr,
	})
}
func middlewareLogging(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		makeLogEntry(c).Info("INCOMING REQUEST")
		return next(c)
	}
}
func errorHandler(err error, c echo.Context) {
	report, ok := err.(*echo.HTTPError)
	if ok {
		report.Message = fmt.Sprintf("http error %d - %v", report.Code, report)
	} else {
		report = echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	makeLogEntry(c).Error(report.Message)
	c.HTML(report.Code, report.Message.(string))
}
type Employee struct {
	No string   `json:"no"   validate:"required"`
	Name string `json:"name" validate:"required,email"`
	Age int     `json:"age"  validate:"gte=17,lte=80"`
}
type Student struct {
	Nisn string `json:"nisn" form:"nisn" query:"nisn"`
	Name string `json:"name" form:"name" query:"name"`
	Age int `json:"age" form:"age" query:"age"`
}
var About = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("about page"))
})
// MIDDLEWARE
func middlewareOne(next echo.HandlerFunc) echo.HandlerFunc  {
	return func(context echo.Context) error {
		fmt.Println("from middleware one")
		return next(context)
	}
}
func middlewareTwo(next echo.HandlerFunc) echo.HandlerFunc {
	return func(context echo.Context) error {
		fmt.Println("from middleware two")
		return next(context)
	}
}
func middlewareSomething(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("from middleware something")
		next.ServeHTTP(w, r)
	})
}