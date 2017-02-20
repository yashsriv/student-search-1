package main

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"

	"github.com/yashsriv/student-search/database"

	"github.com/PuerkitoBio/goquery"
	"gopkg.in/pg.v5"
)

// SafeMap - A Safe Map Construct
type SafeMap struct {
	v   map[string]int
	mux sync.Mutex
}

func (m *SafeMap) Create(key string) {
	log.Println("Creating ", key)
	m.mux.Lock()
	defer m.mux.Unlock()
	_, ok := m.v[key]
	if !ok {
		m.v[key] = 1
	} else {
		log.Fatalf("Key Already Exists: %s\n", key)
	}
}

func (m *SafeMap) Processed(key string) {
	log.Println("Processed ", key)
	m.mux.Lock()
	defer m.mux.Unlock()
	v, ok := m.v[key]
	if !ok {
		log.Fatalf("Key Not Present: %s\n", key)
	} else {
		if v == 1 {
			m.v[key] = v + 1
		} else {
			log.Fatalf("Key: %s State: %d Trying: %d\n", key, v, 1)
		}
	}
}

func (m *SafeMap) Inserted(key string) error {
	log.Println("Inserted ", key)
	m.mux.Lock()
	defer m.mux.Unlock()
	v, ok := m.v[key]
	if !ok {
		log.Fatalf("Key Not Present: %s\n", key)
		return errors.New("Key not Present")
	} else {
		if v == 2 {
			m.v[key] = v + 1
		} else {
			log.Printf("Key: %s State: %d Trying: %d\n", key, v, 2)
			return errors.New("Key already updated")
		}
	}
	return nil
}

func (m *SafeMap) safePrint() {
	for k, v := range m.v {
		fmt.Printf("Key: %s, Value: %d\n", k, v)
	}
}

func (m *SafeMap) Print() {
	m.mux.Lock()
	defer m.mux.Unlock()
	for k, v := range m.v {
		fmt.Printf("Key: %s, Value: %d\n", k, v)
	}
}

// S a safe map for poop
var S = SafeMap{v: make(map[string]int)}

func getAddress(doc *goquery.Document) string {
	body, err := doc.Html()
	if err != nil {
		log.Printf("Error in getAddress: %s\n", err.Error())
		return ""
	}
	if len(strings.Split(body, "Permanent Address :")) < 2 {
		return ""
	}
	address := strings.Split(strings.Split(body, "Permanent Address :")[1], ",")
	length := len(address)
	if len(address) > 2 {
		address = address[length-3 : length-1]
		return fmt.Sprintf("%s, %s", address[0], address[1])
	}
	return ""
}

func fetchStudent(roll string, ch chan database.Student, wg *sync.WaitGroup) {
	defer wg.Done()
	url := fmt.Sprintf("http://oa.cc.iitk.ac.in:8181/Oa/Jsp/OAServices/IITk_SrchRes.jsp?typ=stud&numtxt=%s&sbm=", roll)
	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Printf("Error in fetchStudent readDoc: %s\n", err.Error())
		return
	}
	studentInfo := doc.Find(".TableContent p")
	student := database.Student{}
	student.Roll = roll
	studentInfo.Each(func(i int, s *goquery.Selection) {
		body := s.Text()
		field := strings.Split(strings.TrimSpace(body), ":")
		key := strings.TrimSpace(field[0])
		value := strings.TrimSpace(field[1])
		switch key {
		case "Name":
			student.Name = strings.Title(strings.ToLower(value))
		case "Program":
			student.Program = value
		case "Department":
			student.Dept = strings.Title(strings.ToLower(value))
		case "Hostel Info":
			if len(strings.Split(value, ",")) > 1 {
				student.Hall = strings.Split(value, ",")[0]
				student.Room = strings.Split(value, ",")[1]
			}
		case "E-Mail":
			if len(strings.Split(value, "@")) > 1 {
				student.Username = strings.Split(value, "@")[0]
			}
		case "Blood Group":
			student.BloodGroup = value
		case "Gender":
			if len(strings.Split(value, "\t")) > 1 {
				student.Gender = strings.TrimSpace((strings.Split(value, "\t")[0]))
			}
		default:
			fmt.Printf("%s %s\n", key, value)
		}
	})
	student.Hometown = getAddress(doc)
	S.Processed(roll)
	ch <- student
}

func fetchNums(count int, ch chan database.Student, wg *sync.WaitGroup) {
	defer wg.Done()
	url := fmt.Sprintf("http://oa.cc.iitk.ac.in:8181/Oa/Jsp/OAServices/IITk_SrchStudRoll.jsp?recpos=%d&selstudrol=&selstuddep=&selstudnam=", count)
	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Printf("Error in fetchNums: %s\n", err.Error())
		return
	}
	doc.Find(".TableText a").Each(func(i int, s *goquery.Selection) {
		roll := strings.TrimSpace(s.Text())
		S.Create(roll)
		wg.Add(1)
		go fetchStudent(roll, ch, wg)
	})
}

func main() {

	runtime.GOMAXPROCS(8)

	db := pg.Connect(&pg.Options{
		User: "postgres",
	})

	log.SetPrefix("[student-search] ")

	err := database.CreateStudentSchema(db)
	if err != nil {
		log.Printf("Error in createSchema: %s\n", err.Error())
	}

	batchSize := 540 // Multiple of 12 to prevent repititions
	total := 8010
	log.Println("Starting")
	for j := 0; j < total+1; j += batchSize {
		ch := make(chan database.Student)
		var wg sync.WaitGroup
		var wgdb sync.WaitGroup
		go func() {
			for value := range ch {
				// Add 1 to waitgroup for each go routine launched
				wgdb.Add(1)
				go insertToDatabase(&value, db, &wgdb)
			}
		}()
		for i := j; i < j+batchSize; i += 12 {
			// Add 1 to waitgroup for each go routine launched
			wg.Add(1)
			go fetchNums(i, ch, &wg)
		}
		wg.Wait()
		close(ch)
		log.Println("Closing Channel")
		wgdb.Wait()
		fmt.Printf("Done %d\n", j+batchSize)
	}
	fmt.Println("Done")
	err = db.Close()
	if err != nil {
		log.Println("Error closing db: ", err)
	}
}

func insertToDatabase(value *database.Student, db *pg.DB, wg *sync.WaitGroup) {
	defer wg.Done()
	err := S.Inserted(value.Roll)
	if err == nil {
		err := db.Insert(value)
		if err != nil {
			S.Print()
			log.Fatalf("Error in insert: %s, %v\n", err, value)
		}
	}
}
