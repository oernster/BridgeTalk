// The product's name as the application gives it.
//
// The name is written down once on the Go side and reaches the page through About. The page keeps
// no copy of its own: a fallback here would be a second place the name is kept, which is what lets
// a rename leave a stale one behind. Until About answers the name is empty; each surface showing it
// then says nothing in its place rather than a guess.

import { useEffect, useState } from 'react'
import { api } from './api'

/** useProductName answers the product's name as About gives it; empty until About answers. */
export function useProductName(): string {
  const [name, setName] = useState('')
  useEffect(() => {
    void api.about().then((about) => setName(about?.name ?? ''))
  }, [])
  return name
}
