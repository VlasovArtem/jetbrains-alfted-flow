package reader

import (
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"io/fs"
	"io/ioutil"
	"jetbrains-project-workflow/pkg/service"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var possibleProductNames = []string{"AI", "OC", "CL", "DB", "GO", "IC", "IU", "PS", "PC", "PY", "RD", "RM", "WS", "IE", "MPS", "PE"}

type application struct {
	XMLName   xml.Name  `xml:"application"`
	Component component `xml:"component"`
}

type component struct {
	XMLName xml.Name `xml:"component"`
	Name    string   `xml:"name,attr"`
	Option  option   `xml:"option"`
}

type option struct {
	XMLName xml.Name `xml:"option"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
	Map     XmlMap   `xml:"map"`
}

type XmlMap struct {
	XMLName xml.Name `xml:"map"`
	Entry   []entry  `xml:"entry"`
}

type entry struct {
	XMLName xml.Name `xml:"entry"`
	Key     string   `xml:"key,attr"`
	Value   []value  `xml:"value"`
}

type value struct {
	XMLName               xml.Name              `xml:"value"`
	RecentProjectMetaInfo RecentProjectMetaInfo `xml:"RecentProjectMetaInfo"`
}

type RecentProjectMetaInfo struct {
	XMLName            xml.Name `xml:"RecentProjectMetaInfo"`
	FrameTitle         string   `xml:"frameTitle,attr"`
	ProjectWorkspaceId string   `xml:"projectWorkspaceId,attr"`
	Opened             string   `xml:"opened,attr"`
	Option             options  `xml:"option"`
}

type options []option

type applicationParsedDto struct {
	projectDetails       service.ProjectDetails
	path                 string
	buildTimestamp       int
	projectOpenTimestamp int
	projectBuildDetails  service.ProjectBuildDetails
	opened               bool
}

type project struct {
	recentProjectPath string
	applicationName   string
}

var rootPath string

type buildNumberType []string

func (bn buildNumberType) Len() int {
	return len(bn)
}

func (bn buildNumberType) Less(i, j int) bool {
	return bn[i] > bn[j]
}

func (bn buildNumberType) Swap(i, j int) {
	bn[i], bn[j] = bn[j], bn[i]
}

func ReadProjects(service *service.ProjectService, ignoreProjects map[string]string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	rootPath = fmt.Sprintf("%s/Library/Application Support/JetBrains/", homeDir)
	var projects []project
	var toolboxFolderPath = ""
	err = filepath.Walk(rootPath,
		func(path string, info os.FileInfo, err error) error {
			if info != nil {
				name := info.Name()
				if strings.Contains(name, "Toolbox") {
					toolboxFolderPath = path
				}
				if name == "recentProjects.xml" || name == "recentSolutions.xml" {
					projectName := findProjectName(path)
					if _, exists := ignoreProjects[projectName]; exists {
						return nil
					}

					projects = append(projects, project{
						applicationName:   projectName,
						recentProjectPath: path,
					})
				}
			}
			return nil
		})

	if err != nil {
		return err
	}

	if toolboxFolderPath == "" {
		err = errors.New("Jetbrains Toolbox is not found.")
	}

	jetbrainsAppPath := findJetbrainsAppPath(rootPath + "Toolbox")
	var parsedDTOsFinal []applicationParsedDto

	for _, project := range projects {
		application, err := readRecentFile(project)

		if err != nil {
			return err
		}

		parsedDTOs,  err := application.parseRecentFile(project.applicationName, homeDir)
		if err != nil {
			return err
		}

		parsedDTOsFinal = append(parsedDTOsFinal, parsedDTOs...)
	}

	service.AddProjects(mapToProjectInfos(parsedDTOsFinal, jetbrainsAppPath))

	service.PrepareServices()

	return err
}

func mapToProjectInfos(final []applicationParsedDto, path map[string]string) (projectInfos []service.ProjectInfo) {
	productionCodeToPaths := make(map[string][]string)
	var failedDTOs []applicationParsedDto

	for _, dto := range final {
		path := path[dto.projectBuildDetails.BuildNumber]

		if path != "" {
			projectInfos = append(projectInfos, dto.toProjectInfo(path))
			productionCodeToPaths[dto.projectBuildDetails.ProductionCode] = append(productionCodeToPaths[dto.projectBuildDetails.ProductionCode], path)
		} else {
			failedDTOs = append(failedDTOs, dto)
		}
	}

	for _, dto := range failedDTOs {
		path := productionCodeToPaths[dto.projectBuildDetails.ProductionCode]

		if path != nil {
			sort.Slice(path, func(i, j int) bool {
				switch strings.Compare(path[i], path[j]) {
				case -1:
					return true
				}
				return false
			})

			projectInfos = append(projectInfos, dto.toProjectInfo(path[0]))
		}
	}

	return projectInfos
}

func (d *applicationParsedDto) toProjectInfo(jetbrainsAppPath string) service.ProjectInfo {
	return service.ProjectInfo{
		ProjectDetails:       d.projectDetails,
		Path:                 d.path,
		BuildTimestamp:       d.buildTimestamp,
		ProjectOpenTimestamp: d.projectOpenTimestamp,
		JetbrainsAppPath:     jetbrainsAppPath,
		Valid: true,
		ProjectBuildDetails: d.projectBuildDetails,
		Opened: d.opened,
	}
}

func readRecentFile(p project) (application application, err error) {
	xmlFile, err := os.Open(p.recentProjectPath)
	if err != nil {
		return application, err
	}

	log.Println(fmt.Sprintf("XML File %s successfully opened.", p.recentProjectPath))
	defer xmlFile.Close()

	bytes, err := ioutil.ReadAll(xmlFile)

	if err != nil {
		return application, err
	}

	err = xml.Unmarshal(bytes, &application)

	if err != nil {
		return application, err
	}

	return application, nil
}

func (app *application) parseRecentFile(applicationName string, userHomeDir string) (parsedDTOs []applicationParsedDto, err error) {
	for _, entry := range app.Component.Option.Map.Entry {
		projectPath := entry.Key

		for _, value := range entry.Value {
			recentProjectMetaInfo := value.RecentProjectMetaInfo
			compile := regexp.MustCompile("\\s–\\s.*")
			projectName := strings.TrimSpace(compile.ReplaceAllString(recentProjectMetaInfo.FrameTitle, ""))

			if projectName == "" {
				findName := regexp.MustCompile("(.*/\\..*)")
				projectName = findName.ReplaceAllString(recentProjectMetaInfo.FrameTitle, "")
			}

			buildTimestamp, _ := strconv.Atoi(recentProjectMetaInfo.Option.findFieldValue("buildTimestamp"))
			projectOpenTimestamp, _ := strconv.Atoi(recentProjectMetaInfo.Option.findFieldValue("projectOpenTimestamp"))
			productionCode := recentProjectMetaInfo.Option.findFieldValue("productionCode")
			buildNumber := strings.ReplaceAll(
				recentProjectMetaInfo.Option.findFieldValue("build"),
				productionCode+"-",
				"")

			parsedDTOs = append(parsedDTOs, applicationParsedDto{
				projectDetails: service.ProjectDetails{
					Name:    projectName,
					Project: applicationName,
				},
				path:                 strings.ReplaceAll(projectPath, "$USER_HOME$", userHomeDir),
				buildTimestamp:       buildTimestamp / 1000,
				projectOpenTimestamp: projectOpenTimestamp / 1000,
				projectBuildDetails:  service.ProjectBuildDetails{
					ProductionCode:       productionCode,
					BuildNumber:          buildNumber,
				},
				opened: strings.EqualFold(
					strings.ToLower(value.RecentProjectMetaInfo.Opened),
					"true"),
			})
		}
	}


	//for _, project := range projectInfos {
	//	if !project.Valid {
	//		data := productionCodeToPaths[project.ProductionCode]
	//		if data == nil {
	//			log.Println("Path for the project "+ project.Name + " and product code " + project.ProductionCode + " is not found.")
	//		} else {
	//			sort.Sort(data)
	//			project.JetbrainsAppPath = data[0]
	//			project.Valid = project.JetbrainsAppPath != ""
	//		}
	//	}
	//}

	return parsedDTOs, nil
}

func findJetbrainsAppPath(toolboxPath string) map[string]string {
	buildToAppPath := make(map[string]string)

	filepath.Walk(toolboxPath,
		func(path string, info fs.FileInfo, err error) error {
			if matched, err := regexp.MatchString("^.*\\.app$", info.Name()); err == nil && matched {
				split := strings.Split(filepath.Dir(path), "/")
				buildNumber := split[len(split)-1]
				buildToAppPath[buildNumber] = path
				return filepath.SkipDir
			}
			return nil
		})

	return buildToAppPath
}

func findProjectName(filePath string) string {
	filePathPart := strings.Replace(filePath, rootPath, "", 1)
	compile := regexp.MustCompile("([0-9]+.?)*/.*")
	return compile.ReplaceAllString(filePathPart, "")
}

func (options *options) findFieldValue(fieldName string) string {
	for _, option := range *options {
		if option.Name == fieldName {
			return option.Value
		}
	}
	return ""
}
