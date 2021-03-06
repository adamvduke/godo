package godo

import "fmt"

const (
	actionsBasePath = "v2/actions"

	// ActionInProgress is an in progress action status
	ActionInProgress = "in-progress"

	//ActionCompleted is a completed action status
	ActionCompleted = "completed"
)

// ImageActionsService handles communition with the image action related methods of the
// DigitalOcean API.
type ActionsService struct {
	client *Client
}

type actionsRoot struct {
	Actions []Action `json:"actions"`
}

type actionRoot struct {
	Event Action `json:"action"`
}

// Action represents a DigitalOcean Action
type Action struct {
	ID           int        `json:"id"`
	Status       string     `json:"status"`
	Type         string     `json:"type"`
	StartedAt    *Timestamp `json:"started_at"`
	CompletedAt  *Timestamp `json:"completed_at"`
	ResourceID   int        `json:"resource_id"`
	ResourceType string     `json:"resource_type"`
}

// List all actions
func (s *ActionsService) List() ([]Action, *Response, error) {
	path := actionsBasePath

	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionsRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Actions, resp, err
}

func (s *ActionsService) Get(id int) (*Action, *Response, error) {
	path := fmt.Sprintf("%s/%d", actionsBasePath, id)
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Event, resp, err
}

func (a Action) String() string {
	return Stringify(a)
}
