package gateway

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"goavega-software/cargo/cargo/common"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type AgentDb struct {
}

func openDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./cargo.db?cache=shared")
	db.SetMaxOpenConns(1)
	return db, err
}

func AddContainer(agentId int, containers []common.Container) error {
	if agentId == 0 {
		return sql.ErrNoRows
	}
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()
	containersStr, _ := json.Marshal(containers)
	stmt, err := db.Prepare("update agents set containers = ? where id = ?")
	_, err = stmt.Exec(containersStr, agentId)
	if err != nil {
		return err
	}

	return nil
}

func AddAgent(agent Agent) (int, error) {
	db, err := openDatabase()
	if err != nil {
		return 0, err
	}
	defer db.Close()
	// insert
	stmt, err := db.Prepare("INSERT INTO agents(name, credentials, containers, web, db) values(?,?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	credentials, _ := json.Marshal(agent.Credentials)
	web, _ := json.Marshal(agent.Web)
	database, _ := json.Marshal(agent.Db)
	containers, _ := json.Marshal(agent.Containers)
	res, err := stmt.Exec(agent.Name, string(credentials), containers, web, database)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func GetAgent(id int) (Agent, error) {
	var agent Agent
	agents, err := GetAgents()
	if err != nil {
		return agent, err
	}
	for _, agent := range agents {
		if agent.Id == id {
			return agent, nil
		}
	}
	return agent, sql.ErrNoRows
}

func GetAgents() ([]Agent, error) {
	db, err := openDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query("select * from agents")
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	var id int
	var name string
	var credentialsStr string
	var containersDb sql.NullString
	var containersStr string = ""
	var webDb sql.NullString
	var webStr string = ""
	var databaseDb sql.NullString
	var dbStr string = ""
	agents := make([]Agent, 0)

	for rows.Next() {
		err = rows.Scan(&id, &name, &credentialsStr, &containersDb, &webDb, &databaseDb)
		log.Println(err, " is the error")
		containers := make([]common.Container, 0)
		if containersDb.Valid {
			containersStr = containersDb.String
			log.Println(containersStr, " is the container string")
			json.Unmarshal([]byte(containersStr), &containers)
		}
		web := Web{}
		if webDb.Valid {
			webStr = webDb.String
			json.Unmarshal([]byte(webStr), &web)
		}
		database := Db{}
		if databaseDb.Valid {
			dbStr = databaseDb.String
			json.Unmarshal([]byte(dbStr), &database)
		}

		credentials := Credentials{}
		json.Unmarshal([]byte(credentialsStr), &credentials)
		agent := Agent{Id: id, Name: name, Credentials: credentials, Containers: containers, Web: web, Db: database}
		agents = append(agents, agent)
	}

	return agents, nil
}

func (agentDb *AgentDb) GetContainers() []common.Container {
	containers := make([]common.Container, 0)
	db, err := openDatabase()
	if err != nil {
		return containers
	}
	defer db.Close()
	stmt, _ := db.Prepare("select id, details from containers")
	rows, err := stmt.Query()
	if err != nil {
		return containers
	}
	var jsonString string
	var id int
	for rows.Next() {
		err = rows.Scan(&id, &jsonString)
		if err != nil {
			return containers
		}
		container := common.Container{}
		json.Unmarshal([]byte(jsonString), &container)
		containers = append(containers, container)
	}
	rows.Close()
	stmt.Close()
	return containers
}

func (agentDb *AgentDb) GetContainer(image string, registry string) common.Container {
	var container common.Container
	request := common.Container{Image: image, Registry: registry}
	id := generateId(request)
	db, err := openDatabase()
	if err != nil {
		return container
	}
	defer db.Close()
	log.Println("generated id is", id)
	stmt, err := db.Prepare("select details from containers where id = ?")
	rows, err := stmt.Query(id)
	var jsonString string

	if rows.Next() {
		err = rows.Scan(&jsonString)
		if err != nil {
			return container
		}
	}
	rows.Close()
	stmt.Close()
	container = common.Container{}
	json.Unmarshal([]byte(jsonString), &container)

	return container
}

func (agentDb *AgentDb) AddContainer(container common.Container) (bool, error) {
	hasContainer := false
	db, err := openDatabase()
	if err != nil {
		return hasContainer, err
	}
	defer db.Close()
	id := generateId(container)
	log.Println("generated id is", id)
	stmt, err := db.Prepare("select id from containers where id = ?")
	rows, err := stmt.Query(id)
	if err != nil {
		return hasContainer, err
	}
	var modifySql string
	log.Println("executed query to find")
	if rows.Next() {
		log.Println("executed query to udpate")
		hasContainer = true
		modifySql = "update containers set details=? where id = ?"
		if err != nil {
			return hasContainer, err
		}
	} else {
		log.Println("executed query to insert")
		modifySql = "insert into containers (details, id) values (?,?)"
		if err != nil {
			return hasContainer, err
		}
	}
	rows.Close()
	stmt.Close()
	stmt, err = db.Prepare(modifySql)
	if err != nil {
		return hasContainer, err
	}
	containerStr, _ := json.Marshal(container)
	_, err = stmt.Exec(containerStr, id)

	return hasContainer, err
}

func generateId(container common.Container) string {
	s := container.Registry + container.Image
	h := sha1.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}
