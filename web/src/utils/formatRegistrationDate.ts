const MONTHS_GENITIVE = [
  'января',
  'февраля',
  'марта',
  'апреля',
  'мая',
  'июня',
  'июля',
  'августа',
  'сентября',
  'октября',
  'ноября',
  'декабря',
] as const

/** Читаемая дата регистрации, например «12 июля 2026». */
export function formatRegistrationDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  const day = date.getDate()
  const month = MONTHS_GENITIVE[date.getMonth()]
  const year = date.getFullYear()
  return `${day} ${month} ${year}`
}
