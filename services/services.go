package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-redis/redis"
	pb "github.com/iamgak/grpc_gis/proto"
)

// NewMadinaGisService initializes a new GIS service instance
func NewMadinaGisService(username, password, clientIP, port string, redisClient *redis.Client) *MadinaGisService {
	return &MadinaGisService{
		username: username,
		password: password,
		clientIP: clientIP,
		redis:    redisClient,
		port:     port,
	}
}

type MadinaGisGRPCServer struct {
	pb.UnimplementedMadinaGisServiceServer
	service *MadinaGisService
}

// madinaGisService struct
type MadinaGisService struct {
	username string
	password string
	clientIP string
	redis    *redis.Client
	token    struct {
		mu sync.Mutex
	}
	port string
}

func NewMadinaGisGRPCServer(service *MadinaGisService, client *redis.Client) *MadinaGisGRPCServer {
	return &MadinaGisGRPCServer{
		service: service,
		// redis:   client,
	}
}

// getCachedToken retrieves the token from Redis
func (s *MadinaGisService) getCachedToken() (string, error) {
	token, err := s.redis.Get("madina_gis_token").Result()
	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return token, nil
}

// setCachedToken stores the token in Redis
func (s *MadinaGisService) setCachedToken(token string, expiresAt time.Time) error {
	expiration := time.Until(expiresAt)
	return s.redis.Set("madina_gis_token", token, expiration).Err()
}

func (s *MadinaGisGRPCServer) GetLocationInfo(ctx context.Context, req *pb.LocationRequest) (*pb.LocationResponse, error) {
	apiURL := fmt.Sprintf("https://investment.amana-md.gov.sa/LocationWebService/home/getLocationInfo?longitude=%s&latitude=%s", req.Longitude, req.Latitude)

	data, err := s.service.fetchFromGIS(apiURL)
	if err != nil {
		return nil, err
	}

	fmt.Println("test:1")
	resp := &pb.LocationResponse{Data: make(map[string]string)}
	for k, v := range data {
		resp.Data[k] = fmt.Sprintf("%v", v)
	}
	return resp, nil
}

func (s *MadinaGisGRPCServer) GetParcelStreetData(ctx context.Context, req *pb.LocationRequest) (*pb.ParcelStreetResponse, error) {
	apiURL := fmt.Sprintf("https://geomed.amana-md.gov.sa/arcgis/rest/services/AppVisualDistortion/VisualDistortion/MapServer/1/query?where=1=1&geometry=%s,%s&geometryType=esriGeometryPoint&spatialRel=esriSpatialRelIntersects&outFields=*&returnGeometry=false&f=json", req.Longitude, req.Latitude)
	data, err := s.service.fetchFromGIS(apiURL)
	if err != nil {
		return nil, err
	}

	resp := &pb.ParcelStreetResponse{Data: make(map[string]string)}
	for k, v := range data {
		resp.Data[k] = fmt.Sprintf("%v", v)
	}
	return resp, nil
}

func (s *MadinaGisGRPCServer) GetSatelliteViewData(ctx context.Context, req *pb.Empty) (*pb.SatelliteResponse, error) {
	apiURL := "https://geomed.amana-md.gov.sa/arcgis/rest/services/Hosted/Madinah2020T/MapServer?f=json"
	data, err := s.service.fetchFromGIS(apiURL)
	if err != nil {
		return nil, err
	}

	resp := &pb.SatelliteResponse{Data: make(map[string]string)}
	for k, v := range data {
		resp.Data[k] = fmt.Sprintf("%v", v)
	}
	return resp, nil
}

// ConvertUTMToLatLon converts UTM coordinates to latitude/longitude
func (s *MadinaGisGRPCServer) ConvertUTMToLatLon(ctx context.Context, req *pb.UTMRequest) (*pb.LatLonResponse, error) {
	x, y := req.X, req.Y
	if x == "" || y == "" {
		return nil, fmt.Errorf("error: x and y coordinates are required")
	}

	token, err := s.service.generateToken()
	if err != nil {
		return nil, fmt.Errorf("error: fetching cache data", err)
	}

	params := url.Values{}
	params.Set("f", "json")
	params.Set("inSR", "32637")
	params.Set("outSR", "4326")
	params.Set("geometries", fmt.Sprintf("{\"geometryType\":\"esriGeometryPoint\",\"geometries\":[{\"x\":%s,\"y\":%s}]}", x, y))
	params.Set("token", token)

	apiURL := fmt.Sprintf("https://geomed.amana-md.gov.sa/arcgis/rest/services/Utilities/Geometry/GeometryServer/project?%s", params.Encode())
	fmt.Println(apiURL)
	data, err := s.service.fetchFromGIS(apiURL)
	if err != nil {
		return nil, fmt.Errorf("error: fetching data from gis service", err)
	}
	var result1 struct {
		Geometries []struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"geometries"`
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshaling data map: %w", err)
	}
	err = json.Unmarshal(jsonBytes, &result1)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling GIS response: %w", err)
	}

	fmt.Printf("Parsed Geometries: %+v\n", result1.Geometries)
	result := &pb.LatLonResponse{
		Latitude:  result1.Geometries[0].X,
		Longitude: result1.Geometries[0].Y,
	}
	return result, nil
}

func (s *MadinaGisService) fetchFromGIS(apiURL string) (map[string]interface{}, error) {
	fmt.Println(apiURL)
	token, err := s.generateToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// generateToken generates a new GIS token
func (s *MadinaGisService) generateToken() (string, error) {
	s.token.mu.Lock()
	defer s.token.mu.Unlock()

	cachedToken, err := s.getCachedToken()
	if err == nil && cachedToken != "" {
		log.Println("Using cached GIS token.")
		return cachedToken, nil
	}

	resp, err := http.PostForm("https://geomed.amana-md.gov.sa/portal/sharing/rest/generateToken",
		url.Values{
			"username":   {s.username},
			"password":   {s.password},
			"client":     {s.clientIP},
			"ip":         {s.clientIP},
			"expiration": {"60"},
			"f":          {"json"},
			"referer":    {"https://www.arcgis.com"},
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing token response: %v", err)
	}

	token, ok := result["token"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get token")
	}

	expiresAt := time.Now().Add(time.Minute * 60)
	_ = s.setCachedToken(token, expiresAt)

	return token, nil
}
