import type { Scenario } from './types'

const SCENARIO_VALUES: readonly string[] = [
  'create', 'update', 'delete', 'search', 'list', 'detail', 'export',
]

function isValidScenario(v: string): v is Scenario {
  return SCENARIO_VALUES.includes(v)
}

export class Scenarios extends Array<Scenario> {
  constructor(items?: Scenario[]) {
    super()
    if (items) {
      items.forEach(item => this.push(item))
    }
  }

  has(scenario: Scenario): boolean {
    return this.includes(scenario)
  }

  static from(raw: string | string[]): Scenarios {
    if (typeof raw === 'string') {
      const items = raw.split(';').filter(Boolean)
      const valid = items.filter(isValidScenario)
      if (valid.length !== items.length) {
        const dropped = items.filter(v => !isValidScenario(v))
        console.warn(`[rest-ui] Scenarios.from dropped unknown values: ${dropped.join(', ')}`)
      }
      return new Scenarios(valid)
    }
    const valid = raw.filter(isValidScenario)
    if (valid.length !== raw.length) {
      const dropped = raw.filter(v => !isValidScenario(v))
      console.warn(`[rest-ui] Scenarios.from dropped unknown values: ${dropped.join(', ')}`)
    }
    return new Scenarios(valid)
  }
}
