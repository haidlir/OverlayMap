package app

import (
  "github.com/revel/revel"

  "database/sql"
  "fmt"
  _ "github.com/mattn/go-sqlite3"
  "time"
  "OverlayMap/lib"
)

type projectStatusNull struct {
  Id int
  ProjectName, Description, FileName_001, FileName_002 string
  MaxDist float64	
}

var (
  // AppVersion revel app version (ldflags)
  AppVersion string

  // BuildTime revel app build-time (ldflags)
  BuildTime string
)

var DB *sql.DB
var PathToFile string = "src/OverlayMap/"

func InitDB() {
  revel.INFO.Println("Connecting to DB")
  var err error
  DB, err = sql.Open("sqlite3", PathToFile + "db.db")

  if err != nil {
      revel.INFO.Println("DB Error", err)
  }
  revel.INFO.Println("DB Connected")
}

func init() {
  // Filters is the default set of global filters.
  revel.Filters = []revel.Filter{
    revel.PanicFilter,             // Recover from panics and display an error page instead.
    revel.RouterFilter,            // Use the routing table to select the right Action
    revel.FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
    revel.ParamsFilter,            // Parse parameters into Controller.Params.
    revel.SessionFilter,           // Restore and write the session cookie.
    revel.FlashFilter,             // Restore and write the flash cookie.
    revel.ValidationFilter,        // Restore kept validation errors and save new ones from cookie.
    revel.I18nFilter,              // Resolve the requested language
    HeaderFilter,                  // Add some security based headers
    revel.InterceptorFilter,       // Run interceptors around the action.
    revel.CompressFilter,          // Compress the result.
    revel.ActionInvoker,           // Invoke the action.
  }


  // register startup functions with OnAppStart
  // revel.DevMode and revel.RunMode only work inside of OnAppStart. See Example Startup Script
  // ( order dependent )
  // revel.OnAppStart(ExampleStartupScript)
  revel.OnAppStart(InitDB)
  // revel.OnAppStart(FillCache)
  
  // Custom Init
  revel.OnAppStart(checkJob)
}

// HeaderFilter adds common security headers
// TODO turn this into revel.HeaderFilter
// should probably also have a filter for CSRF
// not sure if it can go in the same filter or not
var HeaderFilter = func(c *revel.Controller, fc []revel.Filter) {
  c.Response.Out.Header().Add("X-Frame-Options", "SAMEORIGIN")
  c.Response.Out.Header().Add("X-XSS-Protection", "1; mode=block")
  c.Response.Out.Header().Add("X-Content-Type-Options", "nosniff")

  fc[0](c, fc[1:]) // Execute the next filter stage.
}

//func ExampleStartupScript() {
//	// revel.DevMod and revel.RunMode work here
//	// Use this script to check for dev mode and set dev/prod startup scripts here!
//	if revel.DevMode == true {
//		// Dev mode
//	}
//}

func checkJob() {
  go func() {
    for {
      // check DB for jobs
      var err error
      lenProjects := -1
      row, err := DB.Query("select COUNT(id) from `project-table` where status isNull;")
      if err != nil {
        revel.ERROR.Println(err)
        return
      }
      for row.Next() {
        err = row.Scan(&lenProjects)
        if err != nil {
          revel.ERROR.Println(err)
        }
      }

      revel.INFO.Println("Project available: ", lenProjects)

      // if empty, wait for 10s
      if lenProjects == 0 {
        time.Sleep(10 * time.Second)
        continue
      }

      //if there is job
      projects, err := DB.Query("select id, `project-name`, description, radius, input_001_filename, input_002_filename from `project-table` where status isNull ORDER BY id ASC;")
      if err != nil {
        revel.ERROR.Println(err)
        return
      }

      jobs := []*projectStatusNull{}

      for projects.Next() {
        var id int
        var projectName, description, fileName_001, fileName_002 string
        var maxDist float64
        err = projects.Scan(&id, &projectName, &description, &maxDist, &fileName_001, &fileName_002)
        if err != nil {
          revel.ERROR.Println(err)
          continue
        }
        jobs = append(jobs, &projectStatusNull{id, projectName, description, fileName_001, fileName_002, maxDist})
      }

      for _, job := range jobs {
        id := job.Id
        projectName := job.ProjectName
        // description := job.Description
        fileName_001 := job.FileName_001
        fileName_002 := job.FileName_002
        maxDist := job.MaxDist

        fileName_001 = fmt.Sprintf("%v-%v", projectName, fileName_001)
        fileName_002 = fmt.Sprintf("%v-%v", projectName, fileName_002)

        err = lib.OverlayFiles(fileName_001, fileName_002, projectName, maxDist, PathToFile)
        // if returns error update with failed: the error explanation
        if err != nil {
          sqlArg := fmt.Sprintf("update `project-table` set status='failed: %v' where id=%v;", err, id)
          _, err = DB.Exec(sqlArg)
          if err != nil {
              revel.ERROR.Println(err)
              continue
          }
          continue
        }
        // otherwise
        output_001_filename := fmt.Sprintf("%v-dataMap.csv", projectName)
        output_002_filename := fmt.Sprintf("%v-mapObj1.csv", projectName)
        output_003_filename := fmt.Sprintf("%v-mapObj2.csv", projectName)
        sqlArg := fmt.Sprintf("update `project-table` set status='done'," +
                              "output_001_filename='%v', output_002_filename='%v'," +
                              "output_003_filename='%v' where id=%v;",
                               output_001_filename, output_002_filename,
                               output_003_filename, id)
        _, err = DB.Exec(sqlArg)
        if err != nil {
            revel.ERROR.Println(err)
            continue
        }
      }
    }
  }()
}
