package model

type ListSchedulingLabelsResponse struct {
	Labels []Label `json:"labels"`
}

type GetServiceSchedulingDetailsResponse struct {
	Current    SchedulingConfig     `json:"current"`
	Selections SchedulingSelections `json:"selections"`
}

type SchedulingConfig struct {
	Labels      []SchedulingLabel      `json:"labels"`
	Node        string                 `json:"node"`
	Tolerations []SchedulingToleration `json:"tolerations"`
}

type SchedulingSelections struct {
	Labels []SchedulingLabel  `json:"labels"`
	Nodes  []NodeBaseInfo     `json:"nodes"`
	Taints TaintForSelectList `json:"taints"`
}

type SchedulingLabel struct {
	Key   string `json:"label_key"`
	Value string `json:"label_value"`
}

type SchdulingNode struct {
	NodeName string `json:"node_name"`
}

type SchedulingToleration struct {
	Key      string `json:"taint_key"`
	Value    string `json:"taint_value"`
	Operator string `json:"op"`
	Effect   string `json:"effect"`
}

type AddServiceSchedulingLabelRequest struct {
	Key   string `json:"label_key"`
	Value string `json:"label_value"`
}

type UpdateServiceSchedulingLabelRequest struct {
	Key   string `json:"label_key"`
	Value string `json:"label_value"`
}

type DeleteServiceSchedulingLabelRequest struct {
	Key string `json:"label_key"`
}

type SetServiceSchedulingNodeRequest struct {
	NodeName string `json:"node_name"`
}

type AddServiceSchedulingTolerationRequest struct {
	Key      string `json:"taint_key"`
	Value    string `json:"taint_value"`
	Operator string `json:"op"`
	Effect   string `json:"effect"`
}

type UpdateServiceSchedulingTolerationRequest struct {
	Key      string `json:"taint_key"`
	Value    string `json:"taint_value"`
	Operator string `json:"op"`
	Effect   string `json:"effect"`
}

type DeleteServiceSchedulingTolerationRequest struct {
	Key string `json:"taint_key"`
}
