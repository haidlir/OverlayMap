package controllers

import (
  "github.com/revel/revel"

  "database/sql"
  "errors"
  "fmt"
  "io/ioutil"
  "math"
  "os"
  "OverlayMap/app"
)

type Project struct {
  Id int64
  ProjectName, Description, InputFileName_001, InputFileName_002 string
  OutputFileName_001, OutputFileName_002, OutputFileName_003 string
  Status string
  MaxDist float64
}

func (p *Project) IsDone() bool {
  if p.Status == "done" {
    return true
  }
  return false }

func (p *Project) IsOnGoing() bool {
  if p.Status == "" {
    return true
  }
  return false }

type App struct {
  *revel.Controller
}

func (c App) Index() revel.Result {
  db := app.DB

  rows, err := db.Query("SELECT * FROM `project-table`")
  if err != nil {
      revel.ERROR.Println(err)
  }

  projects := []*Project{}

  for rows.Next() {
    var id sql.NullInt64
    var projectName, description, inputFileName_001, inputFileName_002 sql.NullString
    var outputFileName_001, outputFileName_002, outputFileName_003 sql.NullString
    var status sql.NullString
    var maxDist sql.NullFloat64
    err = rows.Scan(&id, &projectName, &description, &maxDist, 
                   &inputFileName_001, &inputFileName_002,
                   &outputFileName_001, &outputFileName_002, &outputFileName_003,
                   &status)
    if err != nil {
      revel.ERROR.Println(err)
    }
    projects = append(projects, &Project{id.Int64, projectName.String, description.String, inputFileName_001.String, inputFileName_002.String,
                      outputFileName_001.String, outputFileName_002.String, outputFileName_003.String, status.String, maxDist.Float64})
  }

  c.ViewArgs["projects"] = projects
  return c.Render()
}

func isFileOk(fileByte_001, fileByte_002 []byte) error {
  maxSize := 10 * int(math.Pow10(6)) // 10 MB'
  fmt.Println(len(fileByte_001), len(fileByte_002))
  if len(fileByte_001) < maxSize && len(fileByte_002) < maxSize {
    return nil
  }
  // not OK, better stop
  return errors.New("File size limit exceeded.")
}

func (c App) AddProject() revel.Result {
  db := app.DB

  projectName := c.Params.Form.Get("project-name")
  description := c.Params.Form.Get("description")
  radiusMax := c.Params.Form.Get("radius-max")

  fileName_001 := c.Params.Files["file_001"][0].Filename
  fileName_002 := c.Params.Files["file_002"][0].Filename

  var fileByte_001, fileByte_002 []byte
  c.Params.Bind(&fileByte_001, "file_001")
  c.Params.Bind(&fileByte_002, "file_002")

  var err error
  err = isFileOk(fileByte_001, fileByte_002); if err != nil {
    revel.ERROR.Println(err)
    return c.Redirect(App.Index)
  }

  var sqlArgs string
  sqlArgs = fmt.Sprintf("INSERT INTO `project-table`(`project-name`, description, radius, input_001_filename, input_002_filename) VALUES ('%s','%s','%s','%s','%s')", projectName, description, radiusMax, fileName_001, fileName_002)
  _, err = db.Exec(sqlArgs)
  if err != nil {
      revel.ERROR.Println(err)
      return c.Redirect(App.Index)
  }

  err = ioutil.WriteFile(app.PathToFile+"public/assets/"+projectName+"-"+fileName_001, fileByte_001, 0644)
  if err != nil {
      revel.ERROR.Println(err)
  }

  err = ioutil.WriteFile(app.PathToFile+"public/assets/"+projectName+"-"+fileName_002, fileByte_002, 0644)
  if err != nil {
      revel.ERROR.Println(err)
  }

  return c.Redirect(App.Index)
}

func (c App) DeleteProject() revel.Result {
  db := app.DB

  // Multiple Check-box
  var err error
  err = c.Request.ParseForm()
  if err !=  nil {
    revel.ERROR.Println("Error Delete Project: ", err)
  }
  deleteList := c.Request.Form["delete-list"]
  for _, idDel := range deleteList {
    // get filenames related to the project
    sqlArgs := fmt.Sprintf("select `project-name`, input_001_filename, input_002_filename, " +
                           "output_001_filename, output_002_filename, output_003_filename " +
                           "from `project-table` where id=%v", idDel)
      projects, err := db.Query(sqlArgs)
      if err != nil {
        revel.ERROR.Println(err)
        continue
      }

      deleteFiles := []*string{}
      for projects.Next() {
        var projectName, file1, file2, file3, file4, file5 string
        err = projects.Scan(&projectName, &file1, &file2, &file3, &file4, &file5)
        file1 = fmt.Sprintf("%v-%v", projectName, file1)
        file2 = fmt.Sprintf("%v-%v", projectName, file2)
        if err != nil {
          revel.ERROR.Println(err)
          continue
        }
        deleteFiles = append(deleteFiles, &file1, &file2, &file3, &file4, &file5)
      }

    // delete files
    for _, filename := range deleteFiles {
      if filename == nil {
        continue
      }
      err := os.Remove(app.PathToFile+"public/assets/"+*filename)
      if err != nil {
        revel.ERROR.Println(err)
        continue
      }
    }

    // delete project from datebase
    for _, idDel := range deleteList {
      sqlArgs = fmt.Sprintf("DELETE FROM `project-table` WHERE  `id`='%v';", idDel)
      _, err = db.Exec(sqlArgs)
      if err != nil {
          revel.ERROR.Println(err)
          continue
      }
    }
  }
  return c.Redirect(App.Index)
}