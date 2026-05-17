export class Scenarios extends Array<string> {
  constructor(items?: string[]) {
    super()
    if (items) {
      items.forEach(item => this.push(item))
    }
  }

  has(scenario: string): boolean {
    return this.includes(scenario)
  }

  static from(raw: string | string[]): Scenarios {
    if (typeof raw === 'string') {
      return new Scenarios(raw.split(';').filter(Boolean))
    }
    return new Scenarios(raw)
  }
}
