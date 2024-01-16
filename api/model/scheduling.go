package model

type ListSchedulingLabelsResponse struct {
	Labels []KeyValue `json:"labels"`
}

type GetServiceSchedulingDetailsResponse struct {
	Labels      []SchedulingLabel      `json:"labels"`
	Node        SchdulingNode          `json:"node"`
	Tolerations []SchedulingToleration `json:"tolerations"`
}

type SchedulingLabel struct {
	KeyValue `json:",inline"`
}

type SchdulingNode struct {
	NodeName string `json:"node_name"`
}

type SchedulingToleration struct {
	KeyValue `json:",inline"`
	Operator string `json:"op"`
	Effect   string `json:"effect"`
}

type AddServiceSchedulingLabelRequest struct {
	KeyValue `json:",inline"`
}

type UpdateServiceSchedulingLabelRequest struct {
	KeyValue `json:",inline"`
}

type DeleteServiceSchedulingLabelRequest struct {
	Key string `json:"key"`
}

type SetServiceSchedulingNodeRequest struct {
	NodeName string `json:"node_name"`
}

type AddServiceSchedulingTolerationRequest struct {
	KeyValue `json:",inline"`
	Operator string `json:"op"`
	Effect   string `json:"effect"`
}

type UpdateServiceSchedulingTolerationRequest struct {
	KeyValue `json:",inline"`
	Operator string `json:"op"`
	Effect   string `json:"effect"`
}

type DeleteServiceSchedulingTolerationRequest struct {
	Key string `json:"key"`
}
