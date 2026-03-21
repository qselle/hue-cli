package api

import "context"

type SceneResponse struct {
	Data []Scene `json:"data"`
}

type Scene struct {
	ID       string        `json:"id"`
	Metadata SceneMetadata `json:"metadata"`
	Group    ResourceRef   `json:"group"`
	Status   SceneStatus   `json:"status"`
}

type SceneMetadata struct {
	Name string `json:"name"`
}

type ResourceRef struct {
	RID   string `json:"rid"`
	RType string `json:"rtype"`
}

type SceneStatus struct {
	Active string `json:"active"`
}

type ActivateScenePayload struct {
	Recall SceneRecall `json:"recall"`
}

type SceneRecall struct {
	Action string `json:"action"`
}

func (c *Client) ListScenes(ctx context.Context) ([]Scene, error) {
	var resp SceneResponse
	if err := c.getJSON(ctx, "/resource/scene", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) ActivateScene(ctx context.Context, id string) error {
	payload := ActivateScenePayload{
		Recall: SceneRecall{Action: "active"},
	}
	_, err := c.put(ctx, "/resource/scene/"+id, payload)
	return err
}
