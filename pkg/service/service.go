package service

import (
	"fmt"
	"github.com/pkg/errors"
	"sort"
	"strings"
)

type ProjectInfo struct {
	ProjectDetails
	Path                 string
	BuildTimestamp       int
	ProjectOpenTimestamp int
	JetbrainsAppPath     string
	Valid                bool
	ProjectBuildDetails  ProjectBuildDetails
}

type ProjectBuildDetails struct {
	BuildNumber    string
	ProductionCode string
}

type ProjectDetails struct {
	Name    string
	Project string
}

type SortedByNameAndProjectAndOpenDate []ProjectInfo

type SortedByOpenDate []ProjectInfo

func (s SortedByNameAndProjectAndOpenDate) Len() int {
	return len(s)
}

func (s SortedByNameAndProjectAndOpenDate) Less(i, j int) bool {
	switch strings.Compare(s[i].Project, s[j].Project) {
	case -1:
		return true
	case 1:
		return false
	}
	switch strings.Compare(s[i].Name, s[j].Name) {
	case -1:
		return true
	case 1:
		return false
	}
	return s[i].ProjectOpenTimestamp > s[j].ProjectOpenTimestamp
}

func (s SortedByNameAndProjectAndOpenDate) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (so SortedByOpenDate) Len() int {
	return len(so)
}

func (so SortedByOpenDate) Less(i, j int) bool {
	return so[i].ProjectOpenTimestamp > so[j].ProjectOpenTimestamp
}

func (so SortedByOpenDate) Swap(i, j int) {
	so[i], so[j] = so[j], so[i]
}

type ProjectService struct {
	projectsMap    map[ProjectDetails]ProjectInfo
	sortedProjects []ProjectInfo
}

func New() ProjectService {
	return ProjectService{
		projectsMap: make(map[ProjectDetails]ProjectInfo, 0),
	}
}

func (s *ProjectService) AddProject(projectInfo ProjectInfo) {
	s.projectsMap[projectInfo.ProjectDetails] = projectInfo
}

func (s *ProjectService) AddProjects(projectInfos []ProjectInfo) {
	for _, projectInfo := range projectInfos {
		existingProject, exists := s.projectsMap[projectInfo.ProjectDetails]

		if !exists {
			s.AddProject(projectInfo)
		} else {
			if existingProject.ProjectOpenTimestamp < projectInfo.ProjectOpenTimestamp {
				s.AddProject(projectInfo)
			}
		}
	}
}

func (s *ProjectService) GetProject(name string, project string) (ProjectInfo, error) {
	if application, ok := s.projectsMap[ProjectDetails{
		Project: project,
		Name:    name,
	}]; ok {
		return application, nil
	}
	return ProjectInfo{}, errors.New(fmt.Sprintf("ProjectInfo %s not found", name))
}

func (s *ProjectService) GetProjects() []ProjectInfo {
	return s.sortedProjects
}

func (s *ProjectService) FilterProjects(projectName string) (result []ProjectInfo) {
	lowerProjectName := strings.ToLower(projectName)
	for _, project := range s.sortedProjects {
		if strings.Contains(strings.ToLower(project.Name), lowerProjectName) {
			result = append(result, project)
		}
	}
	return result
}

func (s *ProjectService) PrepareServices() {
	for _, info := range s.projectsMap {
		s.sortedProjects = append(s.sortedProjects, info)
	}
	sort.Sort(SortedByOpenDate(s.sortedProjects))
}
