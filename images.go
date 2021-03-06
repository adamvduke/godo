package godo

// ImagesService handles communication with the image related methods of the
// DigitalOcean API.
type ImagesService struct {
	client *Client
}

// Image represents a DigitalOcean Image
type Image struct {
	ID           int      `json:"id,float64,omitempty"`
	Name         string   `json:"name,omitempty"`
	Distribution string   `json:"distribution,omitempty"`
	Slug         string   `json:"slug,omitempty"`
	Public       bool     `json:"public,omitempty"`
	Regions      []string `json:"regions,omitempty"`
}

type imageRoot struct {
	Image Image
}

type imagesRoot struct {
	Images []Image
}

func (i Image) String() string {
	return Stringify(i)
}

// List all sizes
func (s *ImagesService) List() ([]Image, *Response, error) {
	path := "v2/images"

	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	images := new(imagesRoot)
	resp, err := s.client.Do(req, images)
	if err != nil {
		return nil, resp, err
	}

	return images.Images, resp, err
}
