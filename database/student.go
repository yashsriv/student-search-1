package database

import (
	"fmt"
	"time"

	"gopkg.in/pg.v5"
	"gopkg.in/pg.v5/orm"
)

// Student structure to store student information
type Student struct {
	Roll       string    `sql:",pk" json:"i"`
	Username   string    `json:"u"`
	Name       string    `json:"n"`
	Program    string    `json:"p"`
	Dept       string    `json:"d"`
	Hall       string    `json:"h"`
	Room       string    `json:"r"`
	BloodGroup string    `json:"b"`
	Gender     string    `json:"g"`
	Hometown   string    `json:"a"`
	UpdatedAt  time.Time `json:"-"`
}

func (s Student) String() string {
	return fmt.Sprintf("Student %s %s %s", s.Roll, s.Name, s.Dept)
}

// BeforeInsert hook for Student
func (s *Student) BeforeInsert(db orm.DB) error {
	s.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate hook of Student
func (s *Student) BeforeUpdate(db orm.DB) error {
	s.UpdatedAt = time.Now()
	return nil
}

// CreateStudentSchema creates schema for student model
func CreateStudentSchema(db *pg.DB) error {
	model := &(Student{})
	err := db.CreateTable(model, &orm.CreateTableOptions{})
	return err
}
