import { act, render, screen } from '@testing-library/react'
import { StrictMode } from 'react'
import { describe, expect, it } from 'vitest'
import { fakeClient } from '../test/fake.js'
import { MothProvider } from './context.js'
import { useMoth, useMothUser } from './hooks.js'

function WhoAmI() {
  const user = useMothUser()
  return <div>home of {user?.email ?? 'nobody'}</div>
}

describe('MothProvider', () => {
  it('renders loading, then the login screen, then the app across transitions', async () => {
    const { client } = fakeClient()
    render(
      <MothProvider client={client}>
        <WhoAmI />
      </MothProvider>,
    )
    // Restore in flight: the loading splash.
    expect(screen.getByRole('progressbar')).toBeInTheDocument()
    // Restore lands on signedOut: the built-in login screen.
    expect(
      await screen.findByRole('heading', { name: 'Sign in' }),
    ).toBeInTheDocument()
    expect(screen.queryByText(/home of/)).not.toBeInTheDocument()
    // Sign in: the app renders.
    await act(async () => {
      await client.signIn({ email: 'ada@example.com', password: 'pw' })
    })
    expect(await screen.findByText('home of ada@example.com')).toBeInTheDocument()
    // Sign out: back to the login surface, app unmounted.
    await act(async () => {
      await client.signOut()
    })
    expect(
      await screen.findByRole('heading', { name: 'Sign in' }),
    ).toBeInTheDocument()
    expect(screen.queryByText(/home of/)).not.toBeInTheDocument()
  })

  it('renders children always with requireAuth={false}', async () => {
    const { client } = fakeClient()
    render(
      <MothProvider client={client} requireAuth={false}>
        <WhoAmI />
      </MothProvider>,
    )
    expect(await screen.findByText('home of nobody')).toBeInTheDocument()
    await act(async () => {
      await client.signIn({ email: 'ada@example.com', password: 'pw' })
    })
    expect(await screen.findByText('home of ada@example.com')).toBeInTheDocument()
  })

  it('accepts a custom signedOut node and exposes actions via useMoth', async () => {
    const { client } = fakeClient()
    function SignOutButton() {
      const { signOut, user } = useMoth()
      return (
        <button onClick={() => void signOut()}>bye {user?.displayName}</button>
      )
    }
    render(
      <MothProvider client={client} signedOut={<p>custom door</p>}>
        <SignOutButton />
      </MothProvider>,
    )
    expect(await screen.findByText('custom door')).toBeInTheDocument()
    await act(async () => {
      await client.signIn({ email: 'ada@example.com', password: 'pw' })
    })
    const button = await screen.findByRole('button')
    expect(button).toHaveTextContent('bye Ada')
    await act(async () => {
      button.click()
    })
    expect(await screen.findByText('custom door')).toBeInTheDocument()
  })

  it('survives StrictMode double-mounted effects', async () => {
    const { client } = fakeClient()
    render(
      <StrictMode>
        <MothProvider client={client}>
          <WhoAmI />
        </MothProvider>
      </StrictMode>,
    )
    expect(
      await screen.findByRole('heading', { name: 'Sign in' }),
    ).toBeInTheDocument()
    await act(async () => {
      await client.signIn({ email: 'ada@example.com', password: 'pw' })
    })
    expect(await screen.findByText('home of ada@example.com')).toBeInTheDocument()
    await act(async () => {
      await client.signOut()
    })
    expect(
      await screen.findByRole('heading', { name: 'Sign in' }),
    ).toBeInTheDocument()
  })

  it('throws without exactly one of config or client', () => {
    expect(() =>
      render(<MothProvider>{null}</MothProvider>),
    ).toThrow('exactly one')
  })

  it('scopes the moth theme to moth-owned surfaces only', async () => {
    const { client } = fakeClient()
    const { container } = render(
      <MothProvider client={client}>
        <WhoAmI />
      </MothProvider>,
    )
    // Signed out: the login surface is wrapped in .moth-root.
    await screen.findByRole('heading', { name: 'Sign in' })
    expect(container.querySelector('.moth-root')).not.toBeNull()
    expect(document.head.querySelector('style[data-moth-styles]')).not.toBeNull()
    // Signed in: the app subtree is NOT wrapped.
    await act(async () => {
      await client.signIn({ email: 'ada@example.com', password: 'pw' })
    })
    await screen.findByText('home of ada@example.com')
    expect(container.querySelector('.moth-root')).toBeNull()
  })
})
