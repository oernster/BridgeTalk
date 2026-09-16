// The words a cast row is said in, shared by the recorded voices and the machine voices so both
// kinds name the part and count their figures the same way.

/** casting is the part every cast voice holds, whoever it is. */
const casting = "your ship's voice"

/**
 * castLabel names the act a row offers, for a voice under the name it is shown by.
 *
 * A cast row names the part as well as the fact, because "Grace is cast" says the job
 * has been given out without saying what the job is. The part is the same for every
 * voice. That it does not vary is why it is written here beside
 * the sentence it belongs to rather than carried across with the voice.
 */
export function castLabel(shown: string, cast: boolean): string {
  return cast ? `${shown} is cast as ${casting}` : `Cast ${shown}`
}

/** counted names a number of things, in the singular where there is one. */
export function counted(count: number, one: string, many: string): string {
  return `${count.toLocaleString()} ${count === 1 ? one : many}`
}

/** notYetOn says a kind of voice is not available on a platform yet (FR-817, FR-818). */
export function notYetOn(kind: string, platform: string): string {
  return `${kind} are not available on ${platform} yet.`
}
