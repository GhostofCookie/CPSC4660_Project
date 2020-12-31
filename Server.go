package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	_ "github.com/go-pg/pg"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Server Defines a server object to serve html and handle image uploads and
// database connections.
type Server struct {
	MUX              *http.ServeMux
	dbPgSQL, dbMySQL *sql.DB
}

// NewServer Creates a new server object.
func NewServer() *Server {
	s := &Server{
		MUX: http.NewServeMux(),
	}

	fs := http.FileServer(http.Dir("static"))
	s.MUX.Handle("/static/", http.StripPrefix("/static/", fs))

	return s
}

// HandleHTTP Wrapper for MUX handle func.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.MUX.ServeHTTP(w, r)
}

// HandleHTTP Wrapper for MUX handle func.
func (s *Server) HandleHTTP(p string, f func(w http.ResponseWriter, r *http.Request)) {
	s.MUX.HandleFunc(p, f)
}

// ConnectHTTP Starts up an http server on a given port.
func (s *Server) ConnectHTTP(port int) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	handler := &http.Server{Addr: "10.0.0.184:" + strconv.Itoa(port), Handler: s}
	go func() {
		if err := handler.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	<-stop
}

// BuildHTMLTemplate Creates the HTML required based on submitted files.
func (s *Server) BuildHTMLTemplate(file string, path string, fn func(http.ResponseWriter, *http.Request) interface{}) {
	var tmpl = template.Must(template.ParseFiles(file))
	s.MUX.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if fn != nil {
			// Get the function which defines what we do to the page.
			retval := fn(w, r)

			// Execute our crafted HTML response and submit values to the page.
			tmpl.Execute(w, retval)
		}
	})
}

// ConnectDatabases opens the conneections for all the databases.
func (s *Server) ConnectDatabases() error {
	var err error
	s.dbPgSQL, err = ConnectDatabase(
		"postgres",
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=disable",
		5432)
	if err != nil {
		return err
	}
	s.dbMySQL, err = ConnectDatabase("mysql", "%s:%s@tcp(%s:%d)/%s", 3306)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) StoreImage(driver string, name string, format string, size int, contents []byte) error {
	switch driver {
	case "postgres":
		_, err := s.dbPgSQL.Exec(`INSERT INTO media VALUES ($1, $2, $3, $4)`, name, format, size, contents)
		if err != nil {
			return err
		}
		return nil

	case "mysql":
		_, err := s.dbMySQL.Exec(`INSERT INTO media VALUES (?, ?, ?, ?)`, name, format, size, contents)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("!> No such SQL driver")
}

func (s *Server) GetImage(driver string, name string) (*Image, error) {
	var row *sql.Row
	switch driver {
	case "postgres":
		row = s.dbPgSQL.QueryRow(`SELECT * FROM media WHERE name = $1`, name)
		break
	case "mysql":
		row = s.dbMySQL.QueryRow(`SELECT * FROM media WHERE name = ?`, name)
		break
	}
	var i Image
	err := row.Scan(&i.name, &i.format, &i.size, &i.contents)
	return &i, err
}

func (s *Server) GetListImages() []string {
	images := make([]string, 0)
	sql := func(db *sql.DB) error {
		if db != nil {
			rows, err := db.Query(`SELECT name, format FROM media`)
			if err != nil {
				return err
			}
			if rows != nil {
				for rows.Next() {
					var name, format string
					err = rows.Scan(&name, &format)
					if err != nil {
						break
					}
					images = append(images, name+"."+format)
				}
				return err
			}
			return err
		}
		return errors.New("DB is null")
	}

	err := sql(s.dbPgSQL)
	if err != nil {
		log.Println(err)
		return []string{}
	}

	err = sql(s.dbMySQL)
	if err != nil {
		log.Println(err)
		return []string{}
	}

	encountered := map[string]bool{}
	result := []string{}
	for v := range images {
		if encountered[images[v]] == true {
		} else {
			encountered[images[v]] = true
			result = append(result, images[v])
		}
	}
	images = result
	return images
}

// ConnectDatabase connects a given database of a specific type (driver).
func ConnectDatabase(driver string, url string, port int) (*sql.DB, error) {
	const (
		host   = ""
		user   = ""
		pass   = ""
		dbname = ""
	)

	sqlInfo := fmt.Sprintf(url, user, pass, host, port, dbname)

	db, err := sql.Open(driver, sqlInfo)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
