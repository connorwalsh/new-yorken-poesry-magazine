package types

import (
	"database/sql"
	"path"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
)

type PoetTestSuite struct {
	suite.Suite
	db *sql.DB
}

// run before all tests in this suite begin
func (s *PoetTestSuite) SetupSuite() {
	// create users table (referenced by poets)
	err := CreateUsersTable(s.db)
	if err != nil {
		panic(err)
	}

	// create poets table
	err = CreatePoetsTable(s.db)
	if err != nil {
		panic(err)
	}
}

// run after all tests in this suite have complete
func (s *PoetTestSuite) TearDownSuite() {
	_, err := s.db.Exec(`DROP TABLE IF EXISTS poets CASCADE`)
	if err != nil {
		panic(err)
	}
}

// run specific setups before specific tests
func (s *PoetTestSuite) BeforeTest(suiteName, testName string) {
	var (
		err error
	)

	switch testName {
	// drop users table before create table test to see if it works.
	case "TestCreateTable":
		_, err = s.db.Exec(`DROP TABLE IF EXISTS poets CASCADE`)
		if err != nil {
			panic(err)
		}
	case "TestReadAllPoets":
		_, err = s.db.Exec(`DROP TABLE IF EXISTS poets CASCADE`)
		if err != nil {
			panic(err)
		}

		// create poets table
		err = CreatePoetsTable(s.db)
		if err != nil {
			panic(err)
		}
	}
}

func (s *PoetTestSuite) TestCreateTable() {
	err := CreatePoetsTable(testDB)
	s.NoError(err)
}

func (s *PoetTestSuite) TestCreatePoet() {
	userId := uuid.NewV4().String()
	poetId := uuid.NewV4().String()

	// create user
	user := &User{Username: "3jane", Password: "pwd", Email: "3j4n3@tessier.gov"}
	err := user.Create(userId, s.db)
	s.NoError(err)

	// create poet
	poet := &Poet{
		Designer:    userId,
		Name:        "wintermute",
		Description: "mutator of the immutable",
		ExecPath:    path.Join("/poets/", poetId),
	}

	err = poet.Create(poetId, s.db)
	s.NoError(err)
}

func (s *PoetTestSuite) TestReadPoet() {
	userId := uuid.NewV4().String()
	poetId := uuid.NewV4().String()

	// create user
	user := &User{Username: "hamilton", Password: "pwd", Email: "ijk@quaternion.idk"}
	err := user.Create(userId, s.db)
	s.NoError(err)

	// create poet
	poet := &Poet{
		Designer:    userId,
		Name:        "Chum of Chance",
		Description: "explorer of some other dimensionality",
		ExecPath:    path.Join("/poets/", poetId),
	}

	err = poet.Create(poetId, s.db)
	s.NoError(err)

	expectedPoet := poet

	// read poet
	poet = &Poet{Id: poetId}
	err = poet.Read(s.db)
	s.NoError(err)

	// since there isa problem with the postgres and golang time formats w.r.t.
	// timezones, we will just compoare the formtted times here and nillify the
	// times int he structs -__-
	expectedBirthDate := expectedPoet.BirthDate.Format(time.RFC3339)
	expectedDeathDate := expectedPoet.DeathDate.Format(time.RFC3339)
	birthDate := poet.BirthDate.Format(time.RFC3339)
	deathDate := poet.DeathDate.Format(time.RFC3339)

	s.EqualValues(expectedBirthDate, birthDate)
	s.EqualValues(expectedDeathDate, deathDate)

	expectedPoet.BirthDate = time.Time{}
	expectedPoet.DeathDate = time.Time{}
	poet.BirthDate = time.Time{}
	poet.DeathDate = time.Time{}

	s.EqualValues(expectedPoet, poet)
}

func (s *PoetTestSuite) TestReadAllPoets() {
	poetIds := []string{uuid.NewV4().String(), uuid.NewV4().String(), uuid.NewV4().String()}
	userId := uuid.NewV4().String()

	// create user
	user := &User{Username: "cat-eyed-boy", Password: "pwd", Email: "qt@spooky.jp"}
	err := user.Create(userId, s.db)
	s.NoError(err)

	// create poets
	poets := []*Poet{
		{
			Designer:    userId,
			Name:        "ghostA",
			Description: "haunts shoes",
			ExecPath:    path.Join("/poets/", poetIds[0]),
		},
		{
			Designer:    userId,
			Name:        "ghostB",
			Description: "haunts shoe stores",
			ExecPath:    path.Join("/poets/", poetIds[1]),
		},
		{
			Designer:    userId,
			Name:        "ghostC",
			Description: "isn't a ghost",
			ExecPath:    path.Join("/poets/", poetIds[2]),
		},
	}

	for i := 0; i < len(poetIds); i++ {
		err = poets[i].Create(poetIds[i], s.db)
		s.NoError(err)
	}

	resultPoets, err := ReadPoets(s.db)
	s.NoError(err)
	for j := 0; j < len(resultPoets); j++ {
		// compare formatted string times (since postgres and go have different formats -___-)
		s.EqualValues(
			poets[j].BirthDate.Format(time.RFC3339),
			resultPoets[j].BirthDate.Format(time.RFC3339),
		)
		s.EqualValues(
			poets[j].DeathDate.Format(time.RFC3339),
			resultPoets[j].DeathDate.Format(time.RFC3339),
		)

		resultPoets[j].BirthDate = time.Time{}
		resultPoets[j].DeathDate = time.Time{}
		poets[j].BirthDate = time.Time{}
		poets[j].DeathDate = time.Time{}
	}

	s.EqualValues(poets, resultPoets)
}