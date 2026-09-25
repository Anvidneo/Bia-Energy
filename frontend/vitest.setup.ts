// Runs once per test file. React Testing Library doesn't auto-unmount
// between tests unless something wires this up — without it, renders from
// earlier tests in the same file pile up in document.body and later
// queries like getByText start matching more than one element.
import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'

afterEach(() => {
  cleanup()
})
