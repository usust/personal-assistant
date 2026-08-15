import { http } from './http'
import type { ApiResponse, CaptchaResult, LoginResult } from '@/types/api'

export async function getCaptcha() {
  const { data } = await http.get<ApiResponse<CaptchaResult>>('/captcha')
  return data.data
}

export async function login(username: string, password: string, captchaId: string, captchaCode: string) {
  const { data } = await http.post<ApiResponse<LoginResult>>('/login', {
    username,
    password,
    captchaId,
    captchaCode,
  })
  return data.data
}
