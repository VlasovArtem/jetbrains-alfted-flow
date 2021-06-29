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
	"strconv"
	"strings"
)

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
	Option             options  `xml:"option"`
}

type options []option

type project struct {
	recentProjectPath string
	applicationName   string
}

var rootPath string

func ReadProjects(service *service.ProjectService, ignoreProjects map[string]string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	rootPath = fmt.Sprintf("%s/Library/Application Support/JetBrains/", homeDir)
	var projects []project
	var toolboxFolderPath string = ""
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

	for _, project := range projects {
		application, err := readRecentFile(project)

		if err != nil {
			return err
		}

		projectInfos, err := application.parseRecentFile(project.applicationName, homeDir, jetbrainsAppPath)
		if err != nil {
			return err
		}
		service.AddProjects(projectInfos)
	}

	service.PrepareServices()

	return err
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

func (app *application) parseRecentFile(applicationName string, userHomeDir string, path map[string]string) (projectInfos []service.ProjectInfo, err error) {
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
			buildNumber := strings.ReplaceAll(
				recentProjectMetaInfo.Option.findFieldValue("build"),
				recentProjectMetaInfo.Option.findFieldValue("productionCode")+"-",
				"")

			projectInfos = append(projectInfos, service.ProjectInfo{
				ProjectDetails: service.ProjectDetails{
					Name:    projectName,
					Project: applicationName,
				},
				Path:                 strings.ReplaceAll(projectPath, "$USER_HOME$", userHomeDir),
				BuildTimestamp:       buildTimestamp / 1000,
				ProjectOpenTimestamp: projectOpenTimestamp / 1000,
				JetbrainsAppPath: path[buildNumber],
				Valid: path[buildNumber] != "",
			})
		}
	}

	return projectInfos, nil
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
