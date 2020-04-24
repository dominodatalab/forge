package controllers

type StatusUpdate struct {
	Name          string   `json:"name"`
	ObjectLink    string   `json:"objectLink"`
	PreviousState string   `json:"previousState"`
	CurrentState  string   `json:"currentState"`
	ErrorMessage  string   `json:"errorMessage"`
	ImageURLs     []string `json:"imageURLs"`
}
