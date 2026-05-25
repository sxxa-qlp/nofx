import { traderApi } from './traders'
import { strategyApi } from './strategies'
import { configApi } from './config'
import { dataApi } from './data'

export const api = {
  ...traderApi,
  ...strategyApi,
  ...configApi,
  ...dataApi,
}
