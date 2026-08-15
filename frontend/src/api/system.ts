import { http } from './http'
import type { ApiResponse, HealthStatus } from '@/types/api'

export async function getHealth() {
  const { data } = await http.get<ApiResponse<HealthStatus>>('/health')
  return data.data
}
