package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

	pb "github.com/Asadbe/contactlist/contact-service/proto/task"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	host     = "localhost"
	port     = 5431
	user     = "asadbek"
	password = "1"
	dbname   = "mydb"
)

var err error

// Contact ...
type Contact struct {
	id     int    `db:"serial not null"`
	name   string `db:"not null"`
	number string `db:"not null"`
	age    string `db:"not null"`
	email  string `db:"not null"`
}

// ContactManagerI ...
type ContactManagerI interface {
	Add(*pb.Contact) error
	Update(int64, *pb.Contact) error
	Search(name string) ([]*pb.Contact, error)
	Count() (int64, error)
	Delete(id int64) error
	GetAll() ([]*pb.Contact, error)
}

// TaskManager ...
type TaskManager struct {
	connectDB *sqlx.DB
}
type sqlxDB struct {
	connectDB *sqlx.DB
}

// NewContactManager ...
func NewContactManager() (ContactManagerI, error) {
	cm := sqlxDB{}
	psqlInfo := fmt.Sprintf(`user=%s dbname=%s password=%s`, user, dbname, password)
	cm.connectDB, err = sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &cm, nil
}

// ProtoToStruct ...
func ProtoToStruct(tsk *pb.Contact) Contact {
	var task Contact
	task.name = tsk.Name
	task.email = tsk.Email
	task.number = tsk.Number
	task.age = tsk.Age

	return task
}

func (s *sqlxDB) Add(a *pb.Contact) error {
	insertionQuery := `insert into contacts (name, email, number,age) values ($1, $2, $3,$4)`

	_, err := s.connectDB.Exec(insertionQuery, a.Name, a.Email, a.Number, a.Age)

	if err != nil {
		return err
	}

	return nil
}
func (s *sqlxDB) Count() (int64,error) {
	var a int64
	insertionQuery := `select count(*) from contacts`

	err := s.connectDB.QueryRow(insertionQuery).Scan(&a)
	fmt.Println(a)

	if err != nil {
		return 0,err
	}

	return a, err
}

func (s *sqlxDB) Update(id int64, pb *pb.Contact) error {
	updatingQuery := `update contacts set name=$1,email=$2, number=$3,age=$4
	where id =$5`

	_, err := s.connectDB.Exec(updatingQuery, pb.Name, pb.Email, pb.Number, pb.Age, id)

	if err != nil {
		fmt.Println("Can't update")
		return err
	}

	return nil
}
func (s *sqlxDB) Search(name string) ([]*pb.Contact, error) {
	as := []*pb.Contact{}
	updatingQuery := `select id, name, email, number, age from contacts where name ilike $1`

	rows, err := s.connectDB.Query(updatingQuery, name)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	for rows.Next() {
		ts := &pb.Contact{}
		err = rows.Scan(&ts.Id, &ts.Name, &ts.Email, &ts.Number, &ts.Age)
		if err != nil {
			fmt.Println("Can't scan struct")
			return nil, err
		}
		as = append(as, ts)
	}

	if err != nil {
		fmt.Println("Can't update")
		return nil, err
	}

	return as, nil
}

func (s *sqlxDB) Delete(id int64) error {
	fmt.Println("kevotti", id)
	deletingQuery := `delete from contacts where id=$1;`

	_, err = s.connectDB.Exec(deletingQuery, id)

	if err != nil {
		fmt.Println("Can't delete")
		return err
	}
	return nil
}

func (s *sqlxDB) GetAll() ([]*pb.Contact, error) {
	var (
		tss []*pb.Contact
	)

	tss = []*pb.Contact{}

	listTaskQuery := `select id,name,email,number,age from contacts  `

	rows, err := s.connectDB.Queryx(listTaskQuery)

	if err != nil {
		fmt.Println("Can't print task list")
		return nil, err
	}

	for rows.Next() {
		ts := &pb.Contact{}
		err = rows.Scan(&ts.Id, &ts.Name, &ts.Email, &ts.Number, &ts.Age)
		if err != nil {
			fmt.Println("Can't scan struct")
			return nil, err
		}

		tss = append(tss, ts)
	}
	return tss, nil
}

type service struct {
	tmi ContactManagerI
}

func (s *service) CreateTask(ctx context.Context, req *pb.Contact) (*pb.FlagResponse, error) {
	err := s.tmi.Add(req)
	if err != nil {
		return nil, err
	}
	return &pb.FlagResponse{Flag: true}, err
}

func (s *service) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.FlagResponse, error) {
	err := s.tmi.Update(req.Id, req.Task)
	if err != nil {
		return nil, err
	}
	return &pb.FlagResponse{Flag: true}, nil
}

func (s *service) CountTask(ctx context.Context, req *pb.CountRequest) (*pb.CountTaskResponse, error) {
	counts, err := s.tmi.Count()
	if err != nil {
		return nil, err
	}
	return &pb.CountTaskResponse{Count: counts}, nil
}

func (s *service) SearchTask(ctx context.Context, req *pb.SearchTaskRequest) (*pb.SearchTaskResponse, error) {
	fmt.Println(req.GetTask().GetName())
	tasks, err := s.tmi.Search(req.GetId())
	if err != nil {
		return nil, err
	}
	return &pb.SearchTaskResponse{Tasks: tasks}, nil
}
func (s *service) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.FlagResponse, error) {
	err := s.tmi.Delete(req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.FlagResponse{Flag: true}, err
}

func (s *service) GetAllTasks(ctx context.Context, req *pb.GetAllRequest) (*pb.GetAllResponse, error) {
	tasks, err := s.tmi.GetAll()
	if err != nil {
		return nil, err
	}
	return &pb.GetAllResponse{Tasks: tasks}, nil
}

func main() {
	tm, err := NewContactManager()
	if err != nil {
		log.Fatalf("Cannot connect to database: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()

	pb.RegisterManagingServiceServer(s, &service{tm})

	reflection.Register(s)

	log.Println("Running on port:", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
