//app.go

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strconv"

	"github.com/go-gomail/gomail"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mgo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type App struct {
	DB     *mgo.Database
	Router *mux.Router
}

type Attendee struct {
	FirstName   string        `json:"firstName"`
	LastName    string        `json:"lastName"`
	Email       string        `json:"email"`
	Address     string        `json:"address"`
	City        string        `json:"city"`
	State       string        `json:"state"`
	Zip         string        `json:"zip"`
	HomePhone   string        `json:"homePhone"`
	CellPhone   string        `json:"cellPhone"`
	SoberDate   string        `json:"soberDate"`
	WillChair   bool          `json:"willChair"`
	WillSpeak   bool          `json:"willSpeak"`
	HousingPref []interface{} `json:"HousingPref"`
	RoomatePref string        `json:"roomatePref"`
	COVIDStatus bool          `json:"COVIDStatus"`
	Amount      float64        `json:"amount"`
	Fees        float64        `json:"fees"`
	SelectLabel string        `json:"selectLabel"`
	Validated   bool          `json:"validated"`
	Topics      string        `json:"topics"`
	OrderID     string        `json:"orderID"`
}

type AttendeeInserter struct {
	Insert   string     `json:"insert"`
	Document []Attendee `json:"document"`
}

var attendeeCollection *mgo.Collection

const host = "davidgs.com"
const port = "27017"
const user = "davidgs"
const password = ""

func (a *App) Initialize() error {
	rootPEM, err := ioutil.ReadFile("./combined")
	if err != nil {
		return err
	}
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		return errors.New("failed to parse root certificate")
	}
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{},
		NameToCertificate:  map[string]*tls.Certificate{},
		RootCAs:            roots,
		ServerName:         "davidgs.com",
		InsecureSkipVerify: true,
	}
	connString := fmt.Sprintf("mongodb://%s:%s@%s:%s/?tls=true", user, password, host, port)
	clientOptions := options.Client().ApplyURI(connString)
	clientOptions.TLSConfig = tlsConfig
	client, err := mgo.Connect(context.Background(), clientOptions)
	if err != nil {
		return err
	}
	// Check the connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return err
	}
	fmt.Println("Connected to MongoDB!")
	attendeeCollection = client.Database("Ivoryton-March").Collection("attendees")
	a.Router = mux.NewRouter().StrictSlash(true)
	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{"content-type"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)
	a.Router.Use(cors)
	return nil
}

func (a *App) InitializeRoutes() {
	// a.Router.HandleFunc("/api/Attendees", getAttendees).Methods("GET")
	a.Router.HandleFunc("/api/{db}", setupResponse).Methods("OPTIONS")
	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{"content-type"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)
	a.Router.Use(cors)
	a.Router.HandleFunc("/api/{db}", getAll).Methods("GET")
	a.Router.HandleFunc("/api/{db}/{id}", getByID).Methods("GET")
	a.Router.HandleFunc("/api/{db}", insert).Methods("POST", "OPTIONS")
	// router.HandleFunc("/api/{db}/{id}", deleteByID).Methods("DELETE")
	// router.HandleFunc("/api/{db}/{id}", putByID).Methods("PUT")
	// router.HandleFunc("/api/{db}/{id}", updateEvent).Methods("PUT")
	a.Router.HandleFunc("/api/{db}", getAll).Methods("GET")
	a.Router.HandleFunc("/api/{db}/{id}", getByID).Methods("GET")
	a.Router.PathPrefix("/").Handler(http.FileServer(http.Dir("./blind")))
}

func (a *App) Run(addr string) {
	fmt.Println("Running ... ")
	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{"content-type"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)
	// credentials := handlers.AllowCredentials()
	// methods := handlers.AllowedMethods([]string{"POST", "GET", "OPTIONS", "PUT", "DELETE"})
	// origins := handlers.AllowedOrigins([]string{"http://localhost:3000/*"})
	log.Fatal(http.ListenAndServe(addr, cors(a.Router)))
}

func (a *App) GetCollection(db string) (*mgo.Collection, error) {
	switch db {
	case "attendees":
		return attendeeCollection, nil
	default:
		return nil, errors.New("no db specified")
	}
}

func setupResponse(w http.ResponseWriter, req *http.Request) {
	fmt.Println("setupResponse")
	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func insert(w http.ResponseWriter, r *http.Request) {
	fmt.Println("insert")
	setupResponse(w, r)
	var attendee Attendee
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(body))
	json.Unmarshal(body, &attendee)
	fmt.Println("Attendee: ", attendee)
	fmt.Println("Insert Amount: ", attendee.Amount)
	fmt.Println("Insert Fees: ", attendee.Fees)
	_, err = attendeeCollection.InsertOne(context.Background(), attendee)
	if err != nil {
		fmt.Println(err)
	}

	sendResponse(attendee)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func setTemplate(attendee Attendee) string {
	dat, err := os.ReadFile("./template.html")
	if err != nil {
		fmt.Println(err)
	}
	rep := regexp.MustCompile(`<FirstName>`)
	str := rep.ReplaceAllString(string(dat), attendee.FirstName)
	rep = regexp.MustCompile(`<LastName>`)
	str = rep.ReplaceAllString(str, attendee.LastName)
	rep = regexp.MustCompile(`<Email>`)
	str = rep.ReplaceAllString(str, attendee.Email)
	rep = regexp.MustCompile(`<HomePhone>`)
	str = rep.ReplaceAllString(str, attendee.HomePhone)
	rep = regexp.MustCompile(`<CellPhone>`)
	str = rep.ReplaceAllString(str, attendee.CellPhone)
	rep = regexp.MustCompile(`<Address>`)
	str = rep.ReplaceAllString(str, attendee.Address)
	rep = regexp.MustCompile(`<City>`)
	str = rep.ReplaceAllString(str, attendee.City)
	rep = regexp.MustCompile(`<State>`)
	str = rep.ReplaceAllString(str, attendee.State)
	rep = regexp.MustCompile(`<Zip>`)
	str = rep.ReplaceAllString(str, attendee.Zip)
	rep = regexp.MustCompile(`<SoberDate>`)
	str = rep.ReplaceAllString(str, attendee.SoberDate)
	rep = regexp.MustCompile(`<WillChair>`)
	if attendee.WillChair {
		str = rep.ReplaceAllString(str, "✅")
	} else {
		str = rep.ReplaceAllString(str, "❌")
	}
	rep = regexp.MustCompile(`<WillSpeak>`)
	if attendee.WillSpeak {
		str = rep.ReplaceAllString(str, "✅")
	} else {
		str = rep.ReplaceAllString(str, "❌")
	}
	rep = regexp.MustCompile(`<Amount>`)
	str = rep.ReplaceAllString(str, strconv.FormatFloat(attendee.Amount, 'f', 2, 64) )
	fmt.Println("Amount: ", attendee.Amount)
	fmt.Println("Fees: ", attendee.Fees)
	rep = regexp.MustCompile(`<Fee>`)
	str = rep.ReplaceAllString(str, strconv.FormatFloat(attendee.Fees, 'f', 2, 64) )
	rep = regexp.MustCompile(`<Topics>`)
	str = rep.ReplaceAllString(str, attendee.Topics)
	rep = regexp.MustCompile(`<RoomatePref>`)
	str = rep.ReplaceAllString(str, attendee.RoomatePref)
	rep = regexp.MustCompile(`<COVIDStatus>`)
	if attendee.COVIDStatus {
		str = rep.ReplaceAllString(str, "✅")
	} else {
		str = rep.ReplaceAllString(str, "❌")
	}
	for ind, house := range attendee.HousingPref {
		rep = regexp.MustCompile(`<Housing\[` + strconv.Itoa(ind) + `\]>`)
		str = rep.ReplaceAllString(str, house.(string))
	}
	rep = regexp.MustCompile(`<OrderID>`)
	str = rep.ReplaceAllString(str, attendee.OrderID)
	return str
}

func sendResponse(attendee Attendee) {
	fmt.Println("sendResponse")
	template := setTemplate(attendee)
	options := sendOptions{
			To: 		attendee.Email,
			Subject: 	"Ivoryton New Beginnings Conference Registration",
		}

	smtpConfig := smtpAuthentication{
		Server:         "mail.davidgs.com",
		Port:           587,
		SenderEmail:    "davidgs@davidgs.com",
		SenderIdentity: "Ivoryton Conference Committee",
		SMTPPassword:   "Toby66.Mime!",
		SMTPUser:       "davidgs",
	}
	fmt.Println(smtpConfig)
	err := send(smtpConfig, options, template)
	if err != nil {
		fmt.Println(err)
	}
}

func getAttendees(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	fmt.Println("attendees")
	opts := options.Find().SetSort(bson.D{{"Name", 1}})
	cursor, err := attendeeCollection.Find(context.Background(), bson.D{{}}, opts)
	if err != nil {
		http.Error(w, "Fatal Error", 500)
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())
	attendees := []Attendee{}
	if err = cursor.All(context.Background(), &attendees); err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attendees)
}

func deleteEvent(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
}

func getAll(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	db := mux.Vars(r)["db"]
	fmt.Println("getAll() Using DB: ", db)
	col, err := getCollection(db)
	if err != nil {
		http.Error(w, "Fatal Error: "+err.Error(), 500)
		log.Fatal(err)
	}
	opts := options.Find().SetSort(bson.D{{"start", 1}})
	cursor, err := col.Find(context.Background(), bson.D{{}}, opts)
	if err != nil {
		http.Error(w, "Fatal Error: "+err.Error(), 500)
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())
	if db == "attendees" {
		fmt.Println("fetching attendees")
		attendees := []Attendee{}
		if err = cursor.All(context.Background(), &attendees); err != nil {
			log.Fatal(err)
		}
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(attendees)
	}
}

func getByID(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	fmt.Println("Getting an ID")
	db := mux.Vars(r)["db"]
	fmt.Println("getByID() Using DB: ", db)
	eventID := mux.Vars(r)["id"]
	fmt.Println("Looking for ID: ", eventID)
	obID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		http.Error(w, "Fatal Error", 500)
		log.Fatal("ObjectIDFromHex(): ", err)
	}
	col, err := getCollection(db)
	if err != nil {
		http.Error(w, "Fatal Error", 500)
		log.Fatal("getCollection(): ", err)
	}
	curs, err := col.Find(context.TODO(), bson.M{"_id": obID})
	if err != nil {
		http.Error(w, "Fatal Error", 500)
		log.Fatal("Find(): ", err)
	}
	defer curs.Close(context.TODO())
	if db == "Attendees" {
		fmt.Println("Attendees")
		var attendee = []Attendee{}
		err = curs.All(context.TODO(), &attendee)
		if err != nil {
			http.Error(w, "Fatal Error", 500)
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(attendee[0])
	} else {
		http.Error(w, "Fatal Error", 500)
		log.Fatal(errors.New(db + " is not a valid database"))
	}
}

func getCollection(db string) (*mgo.Collection, error) {
	switch db {
	case "attendees":
		return attendeeCollection, nil
	default:
		return nil, errors.New("no db specified")
	}
}

type smtpAuthentication struct {
	Server         string
	Port           int
	SenderEmail    string
	SenderIdentity string
	SMTPUser       string
	SMTPPassword   string
}

// sendOptions are options for sending an email
type sendOptions struct {
	To      string
	Subject string
}

func send(smtpConfig smtpAuthentication, options sendOptions, htmlBody string) error {

	if smtpConfig.Server == "" {
		return errors.New("SMTP server config is empty")
	}
	if smtpConfig.Port == 0 {
		return errors.New("SMTP port config is empty")
	}
	if smtpConfig.SMTPUser == "" {
		return errors.New("SMTP user is empty")
	}
	if smtpConfig.SenderIdentity == "" {
		return errors.New("SMTP sender identity is empty")
	}
	if smtpConfig.SenderEmail == "" {
		return errors.New("SMTP sender email is empty")
	}
	if options.To == "" {
		return errors.New("no receiver emails configured")
	}
	from := mail.Address{
		Name:    smtpConfig.SenderIdentity,
		Address: smtpConfig.SenderEmail,
	}
	m := gomail.NewMessage()
	m.SetHeader("From", from.String())
	m.SetHeader("To", options.To)
	m.SetHeader("Subject", options.Subject)
	m.SetHeader("MIME-Version", "1.0")
	m.SetHeader("Content-Type", "text/html; charset=\"utf-8\"")
	m.SetBody("text/html", htmlBody)
	d := gomail.NewDialer(smtpConfig.Server, smtpConfig.Port, smtpConfig.SMTPUser, smtpConfig.SMTPPassword)
	return d.DialAndSend(m)
}
