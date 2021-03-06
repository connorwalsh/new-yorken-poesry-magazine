package types

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/connorwalsh/new-yorken-poesry-magazine/server/consts"
	"github.com/connorwalsh/new-yorken-poesry-magazine/server/utils"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
)

// User is a user of the New Yorken Poesry Magazine. This entity
// has an account associated with it and is able to create and
// update poets. A user must have a valid email address in order
// to register for the site.
type User struct {
	Id                 string  `json:"id"`
	Username           string  `json:"username"`
	Password           string  `json:"password,omitempty"`
	Email              string  `json:"email"`
	Poets              []*Poet `json:"poets,omitempty"`
	EmailNotifications bool    `json:"emailNotifications"`
}

// this struct is strictly for extracting possibly null valued
// fields from the database -___-
// we will only use this struct if we are OUTER JOINING poets on
// another table (e.g. users, since some users might not have poets)
// TODO (cw|9.20.2018) figure out a better way to do this...
type UserNullable struct {
	Id                 sql.NullString
	Username           sql.NullString
	Password           sql.NullString
	Email              sql.NullString
	EmailNotifications sql.NullBool
}

func (un *UserNullable) Convert() *User {
	return &User{
		Id:                 un.Id.String,
		Username:           un.Username.String,
		Password:           un.Password.String,
		Email:              un.Email.String,
		EmailNotifications: un.EmailNotifications.Bool,
	}
}

// The parameters which should be supplied from the session context
// in order to properly validate the user.
type UserValidationParams struct {
	CurrentUserID string
}

func (u *User) Validate(action string, params ...UserValidationParams) error {
	var (
		err error
	)

	// make sure id, if not an empty string, is a uuid
	if !utils.IsValidUUIDV4(u.Id) && u.Id != "" {
		return fmt.Errorf("User Id must be a valid uuid, given %s", u.Id)
	}

	// perform validation on a per action basis
	switch action {
	case consts.LOGIN:
		// there must be a username and a password
		if u.Username == "" {
			return fmt.Errorf("No username provided.")
		}

		if u.Password == "" {
			return fmt.Errorf("No password provided.")
		}
	case consts.CREATE:
		// on registration, the username, password, and email must be provided
		if u.Username == "" {
			return fmt.Errorf("No username provided.")
		}
		if u.Password == "" {
			return fmt.Errorf("No password provided.")
		}
		if u.Email == "" {
			return fmt.Errorf("No email address provided.")
		}

		// validate username and email
		err = utils.ValidateUsername(u.Username)
		if err != nil {
			return err
		}
		err = utils.ValidateEmail(u.Email)
		if err != nil {
			return err
		}
	case consts.UPDATE:
		// TODO validate user updates
		fallthrough
	case consts.DELETE:
		if len(params) == 0 {
			return fmt.Errorf(
				"validation parameters must be provided for %s action",
				action,
			)
		}

		// a user can only delete themselves!
		if u.Id != params[0].CurrentUserID {
			return fmt.Errorf(
				"I'm sorry, but what are you trying to do? You can't %s other users...",
				action,
			)
		}
	default:
		// only ensure that the id is present
		// this aplies to the READ and DELETE cases
		// we minimally need the Id to exist in these cases
		if u.Id == "" {
			return fmt.Errorf(
				"User Id is a required field to fulfill a %s",
				action,
			)
		}
	}

	return nil
}

// sanitizes User data before it is transmitted over the wire.
//
// this is important to remove sensitive data like passwords.
//
func (u *User) Sanitize() {
	// replace password with asterisks
	u.Password = `*************`
}

/*
   db methods
*/

// package level globals for storing prepared sql statements
var (
	ErrorUserDoesNotExist = errors.New("user does not exist")

	userAuthStmt                 *sql.Stmt
	userCreateStmt               *sql.Stmt
	userReadStmt                 *sql.Stmt
	userReadAllStmt              *sql.Stmt
	userDeleteStmt               *sql.Stmt
	userReadWithEmailAddressStmt *sql.Stmt

	poetOfUserReadStmt *sql.Stmt
)

// TODO refactor this so that is doesn't need a reciever
// aka CreateUsersTable(...)
func CreateUsersTable(db *sql.DB) error {
	mkTableStmt := `CREATE TABLE IF NOT EXISTS users (
		          id UUID NOT NULL UNIQUE,
                          username VARCHAR(255) NOT NULL UNIQUE,
                          password VARCHAR(255) NOT NULL,
                          salt UUID NOT NULL,
                          email VARCHAR(255) NOT NULL UNIQUE,
                          emailNotifications BOOL NOT NULL,
                          deleted BOOL NOT NULL DEFAULT false,
		          PRIMARY KEY (id)
	)`

	_, err := db.Exec(mkTableStmt)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) Create(db *sql.DB) error {
	var (
		hashedPassword string
		salt           string
		err            error
	)

	// we assume that all validation/sanitization has already been called

	// assume id have been assigned.

	// generate salt for password
	salt = uuid.NewV4().String()

	// salt password
	hashedPassword = utils.SaltPassword(u.Password, salt)

	// prepare statement if not already done so.
	if userCreateStmt == nil {
		// create statement
		stmt := `INSERT INTO users (
                           id, username, password, salt, email, emailNotifications
                         ) VALUES ($1, $2, $3, $4, $5, $6)`
		userCreateStmt, err = db.Prepare(stmt)
		if err != nil {
			return err
		}
	}

	_, err = userCreateStmt.Exec(
		u.Id,
		u.Username,
		hashedPassword,
		salt,
		u.Email,
		u.EmailNotifications,
	)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) Authenticate(db *sql.DB) error {
	var (
		hashedPassword string
		salt           string
		err            error
	)

	if userAuthStmt == nil {
		// auth stmt
		stmt := `SELECT id, password, salt FROM users WHERE username = $1`
		userAuthStmt, err = db.Prepare(stmt)
		if err != nil {
			return err
		}
	}

	// assume that auth validation for user has been performed

	// run the prepared stmt over args (username)
	err = userAuthStmt.
		QueryRow(u.Username).
		Scan(&u.Id, &hashedPassword, &salt)
	switch {
	case err == sql.ErrNoRows:
		return fmt.Errorf("incorrect username or password")
	case err != nil:
		return err
	}

	// hash provided user password
	passwd := utils.SaltPassword(u.Password, salt)

	// ensure that our hashed provided password matches our hashed saved password
	if passwd != hashedPassword {
		// oops, wrong password
		return fmt.Errorf("incorrect username or password")
	}

	return nil
}

func (u *User) Read(db *sql.DB) error {
	var (
		err error
	)

	// prepare statement if not already done so.
	if userReadStmt == nil {
		// read statement
		stmt := `
                         SELECT u.id, username, password, email, emailNotifications,
                                p.id, p.name, p.birthDate, p.deathDate, p.description,
                                p.language, p.path
                         FROM users u
                         LEFT OUTER JOIN poets p
                         ON (u.id = p.designer AND p.deleted = false)
                         WHERE u.id = $1 AND u.deleted = false
                `
		userReadStmt, err = db.Prepare(stmt)
		if err != nil {
			return err
		}
	}

	// make sure user Id is actually populated

	// run prepared query over arguments
	rows, err := userReadStmt.Query(u.Id)
	if err != nil {
		return err
	}

	// initialize poets slice
	u.Poets = []*Poet{}

	nRows := 0
	defer rows.Close()
	for rows.Next() {
		poetNullable := &PoetNullable{}
		err = rows.Scan(
			&u.Id,
			&u.Username,
			&u.Password,
			&u.Email,
			&u.EmailNotifications,
			&poetNullable.Id,
			&poetNullable.Name,
			&poetNullable.BirthDate,
			&poetNullable.DeathDate,
			&poetNullable.Description,
			&poetNullable.Language,
			&poetNullable.Path,
		)
		if err != nil {
			return err
		}

		// append non-null poets to list
		if poetNullable.Id.Valid {
			u.Poets = append(u.Poets, poetNullable.Convert())
		}

		nRows += 1
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// if there are no rows, we have an non-existent user
	if nRows == 0 {
		return fmt.Errorf("user with id %s does not exist.", u.Id)
	}

	// TODO ensure that we only allow reading of passwords if the user making the
	// request is the user being read.

	return nil
}

func (u *User) Update(db *sql.DB) error {
	// TODO

	return nil
}

func (u *User) Delete(db *sql.DB) error {
	var (
		err error
	)

	// prepare statement if not already done so.
	if userDeleteStmt == nil {
		// delete statement
		stmt := `DELETE FROM users WHERE id = $1`
		userDeleteStmt, err = db.Prepare(stmt)
		if err != nil {
			return err
		}
	}

	_, err = userDeleteStmt.Exec(u.Id)
	if err != nil {
		return err
	}

	return nil
}

func GetUserWithEmailAddress(email string, db *sql.DB) (*User, error) {
	var (
		user User
		err  error
	)

	if userReadWithEmailAddressStmt == nil {
		stmt := `
                    SELECT u.id, username, email, emailNotifications
                    FROM users u
                    WHERE u.email = $1
                `
		userReadWithEmailAddressStmt, err = db.Prepare(stmt)
		if err != nil {
			return &user, nil
		}
	}

	// run the prepared stmt over args (username)
	err = userReadWithEmailAddressStmt.
		QueryRow(email).
		Scan(&user.Id, &user.Username, &user.Email, &user.EmailNotifications)
	switch {
	case err == sql.ErrNoRows:
		return &user, ErrorUserDoesNotExist
	case err != nil:
		return &user, err
	}

	return &user, nil
}

func ReadUsers(db *sql.DB) ([]*User, error) {
	var (
		users = []*User{}
		err   error
	)

	// prepare statement if not already done so.
	if userReadAllStmt == nil {
		// readAll statement
		// TODO pagination
		stmt := `
                    SELECT u.id, username, email, emailNotifications,
                           p.id, name, birthDate, deathDate, description,
                           language, programFileName, parameterFileName,
                           parameterFileIncluded, path
                    FROM users u
                    LEFT OUTER JOIN poets p
                    ON (u.id = p.designer AND p.deleted = false)
                    WHERE u.deleted = false
                `
		userReadAllStmt, err = db.Prepare(stmt)
		if err != nil {
			return users, nil
		}
	}

	rows, err := userReadAllStmt.Query()
	if err != nil {
		return users, err
	}

	defer rows.Close()
	for rows.Next() {
		user := &User{}
		poetNullable := &PoetNullable{}
		err = rows.Scan(
			&user.Id,
			&user.Username,
			&user.Email,
			&user.EmailNotifications,
			&poetNullable.Id,
			&poetNullable.Name,
			&poetNullable.BirthDate,
			&poetNullable.DeathDate,
			&poetNullable.Description,
			&poetNullable.Language,
			&poetNullable.ProgramFileName,
			&poetNullable.ParameterFileName,
			&poetNullable.ParameterFileIncluded,
			&poetNullable.Path,
		)
		if err != nil {
			return users, err
		}

		// check to see if poetNullable is null...
		if !poetNullable.Id.Valid {
			// if the poet is null, then there must be only one user
			// so add this user and no poets.
			users = append(users, user)

			continue
		}

		// cool, the poet is not null
		poet := poetNullable.Convert()
		poet.Designer.Id = user.Id
		poet.Designer.Username = user.Username
		poet.Designer.Email = user.Email
		poet.Designer.EmailNotifications = user.EmailNotifications

		if len(users) != 0 && user.Id == users[len(users)-1].Id {
			// consolidate poets into one slice according to user
			users[len(users)-1].Poets = append(
				users[len(users)-1].Poets,
				poet,
			)
		} else {
			// append scanned user into list of all users
			user.Poets = []*Poet{poet}
			users = append(users, user)
		}
	}
	if err := rows.Err(); err != nil {
		return users, err
	}

	return users, nil
}
