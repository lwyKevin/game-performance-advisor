package system

type Info struct {
	OS           string   `json:"os"`
	CPUModel     string   `json:"cpu_model"`
	GPUModels    []string `json:"gpu_models"`
	RAMBytes     uint64   `json:"ram_bytes"`
	RAMGB        float64  `json:"ram_gb"`
	Resolution   string   `json:"resolution"`
	RefreshHz    int      `json:"refresh_hz"`
	IsAdmin      bool     `json:"is_admin"`
	CollectionOK bool     `json:"collection_ok"`
}
