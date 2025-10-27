const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

class APIService {
  async get(endpoint) {
    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error(`API GET error for ${endpoint}:`, error);
      throw error;
    }
  }

  async post(endpoint, data = {}) {
    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error(`API POST error for ${endpoint}:`, error);
      throw error;
    }
  }

  // Service endpoints
  async getServices(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const endpoint = `/services${queryString ? `?${queryString}` : ''}`;
    return this.get(endpoint);
  }

  async getServiceHistory(serviceId, params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const endpoint = `/services/${serviceId}/history${queryString ? `?${queryString}` : ''}`;
    return this.get(endpoint);
  }

  async triggerScrape(serviceName) {
    return this.post(`/scrape/${serviceName}`);
  }

  async getScraperStatus() {
    return this.get('/scraper/status');
  }

  async getHealth() {
    return this.get('/health');
  }

  async uploadNetflixCSV(file) {
    try {
      const formData = new FormData();
      formData.append('csv', file);

      const response = await fetch(`${API_BASE_URL}/upload/netflix`, {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('CSV upload error:', error);
      throw error;
    }
  }
}

export default new APIService();
