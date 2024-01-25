package handler

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
)

func (s *ServiceAction) GetServiceSchedulingDetails(serviceID string) (*model.GetServiceSchedulingDetailsResponse, error) {
	var result model.GetServiceSchedulingDetailsResponse
	labels, _ := db.GetManager().TenantEnvServiceSchedulingLabelDao().ListServiceSchedulingLabels(serviceID)
	for _, label := range labels {
		result.Current.Labels = append(result.Current.Labels, model.SchedulingLabel{
			Key:   label.Key,
			Value: label.Value,
		})
	}

	node, _ := db.GetManager().TenantEnvServiceSchedulingNodeDao().GetServiceSchedulingNode(serviceID)
	if node != nil {
		result.Current.Node = node.NodeName
	}

	tolerations, _ := db.GetManager().TenantEnvServiceSchedulingTolerationDao().ListServiceSchedulingTolerations(serviceID)
	for _, toleration := range tolerations {
		result.Current.Tolerations = append(result.Current.Tolerations, model.SchedulingToleration{
			Key:      toleration.Key,
			Value:    toleration.Value,
			Operator: toleration.Operator,
			Effect:   toleration.Effect,
		})
	}

	lslr, _ := GetSchedulingHandler().ListSchedulingLabels()
	if lslr != nil {
		for _, label := range lslr.Labels {
			result.Selections.Labels = append(result.Selections.Labels, model.SchedulingLabel{
				Key:   label.Key,
				Value: label.Value,
			})
		}
	}

	lsnr, _ := GetSchedulingHandler().ListSchedulingNodes()
	if lsnr != nil {
		result.Selections.Nodes = lsnr.Nodes
	}

	lstr, _ := GetSchedulingHandler().ListSchedulingTaints()
	if lstr != nil {
		result.Selections.Taints = lstr.Taints
	}

	return &result, nil
}

func (s *ServiceAction) AddServiceSchedulingLabel(serviceID string, req *model.AddServiceSchedulingLabelRequest) error {
	if req == nil {
		return nil
	}

	if req.Key == "" {
		return fmt.Errorf("调度标签 key 不能为空！")
	}

	get, err := db.GetManager().TenantEnvServiceSchedulingLabelDao().GetServiceSchedulingLabelByKey(serviceID, req.Key)
	if err == nil && get != nil && get.ID > 0 {
		return fmt.Errorf("调度标签 %s 已存在！", req.Key)
	}

	label := dbmodel.TenantEnvServiceSchedulingLabel{
		ServiceID: serviceID,
		Key:       req.Key,
		Value:     req.Value,
	}
	err = db.GetManager().TenantEnvServiceSchedulingLabelDao().AddModel(&label)
	if err != nil {
		return errors.New("设置调度标签失败！")
	}
	return nil
}

func (s *ServiceAction) UpdateServiceSchedulingLabel(serviceID string, req *model.UpdateServiceSchedulingLabelRequest) error {
	if req == nil {
		return nil
	}

	if req.Key == "" {
		return fmt.Errorf("调度标签 key 不能为空！")
	}

	label, _ := db.GetManager().TenantEnvServiceSchedulingLabelDao().GetServiceSchedulingLabelByKey(serviceID, req.Key)
	if label == nil || label.ID <= 0 {
		return fmt.Errorf("调度标签 %s 不存在！", req.Key)
	}

	label.Value = req.Value
	err := db.GetManager().TenantEnvServiceSchedulingLabelDao().UpdateModel(label)
	if err != nil {
		return errors.New("设置调度标签失败！")
	}
	return nil
}

func (s *ServiceAction) DeleteServiceSchedulingLabel(service string, req *model.DeleteServiceSchedulingLabelRequest) error {
	return db.GetManager().TenantEnvServiceSchedulingLabelDao().DeleteModel(service, req.Key)
}

func (s *ServiceAction) SetServiceSchedulingNode(serviceID string, req *model.SetServiceSchedulingNodeRequest) error {
	if req == nil {
		return nil
	}

	err := db.GetManager().TenantEnvServiceSchedulingNodeDao().DeleteModel(serviceID, req.NodeName)
	if err != nil {
		logrus.Errorf("delete service scheduling node failure, error: %v", err)
	}

	if req.NodeName == "" {
		return nil
	}

	node := dbmodel.TenantEnvServiceSchedulingNode{
		ServiceID: serviceID,
		NodeName:  req.NodeName,
	}
	err = db.GetManager().TenantEnvServiceSchedulingNodeDao().AddModel(&node)
	if err != nil {
		return errors.New("设置调度节点失败！")
	}
	return nil
}

func (s *ServiceAction) AddServiceSchedulingToleration(serviceID string, req *model.AddServiceSchedulingTolerationRequest) error {
	if req == nil {
		return nil
	}

	get, err := db.GetManager().TenantEnvServiceSchedulingTolerationDao().GetServiceSchedulingTolerationByKey(serviceID, req.Key)
	if err == nil && get != nil && get.ID > 0 {
		return fmt.Errorf("设置调度容忍冲突！")
	}

	toleration := dbmodel.TenantEnvServiceSchedulingToleration{
		ServiceID: serviceID,
		Key:       req.Key,
		Operator:  req.Operator,
		Value:     req.Value,
		Effect:    req.Effect,
	}
	err = db.GetManager().TenantEnvServiceSchedulingTolerationDao().AddModel(&toleration)
	if err != nil {
		return errors.New("设置调度容忍失败！")
	}
	return nil
}

func (s *ServiceAction) UpdateServiceSchedulingToleration(serviceID string, req *model.UpdateServiceSchedulingTolerationRequest) error {
	if req == nil {
		return nil
	}

	toleration, _ := db.GetManager().TenantEnvServiceSchedulingTolerationDao().GetServiceSchedulingTolerationByKey(serviceID, req.Key)
	if toleration == nil || toleration.ID <= 0 {
		return fmt.Errorf("设置调度容忍不存在！")
	}

	toleration.Operator = req.Operator
	toleration.Value = req.Value
	toleration.Effect = req.Effect

	err := db.GetManager().TenantEnvServiceSchedulingTolerationDao().UpdateModel(toleration)
	if err != nil {
		return errors.New("设置调度容忍失败！")
	}
	return nil
}

func (s *ServiceAction) DeleteServiceSchedulingToleration(service string, req *model.DeleteServiceSchedulingTolerationRequest) error {
	return db.GetManager().TenantEnvServiceSchedulingTolerationDao().DeleteModel(service, req.Key)
}
