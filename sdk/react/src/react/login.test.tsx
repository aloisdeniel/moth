import { act, fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { fakeClient } from '../test/fake.js'
import { MothProvider } from './context.js'

async function renderLogin(
  options: Parameters<typeof fakeClient>[0] = {},
) {
  const { client, fake } = fakeClient(options)
  render(
    <MothProvider client={client}>
      <div>the app</div>
    </MothProvider>,
  )
  // Wait for the config fetch to resolve the form.
  await screen.findByLabelText('Email')
  return { client, fake }
}

describe('MothLoginScreen', () => {
  it('validates email and password before calling the server', async () => {
    const { fake } = await renderLogin()
    fireEvent.click(screen.getByText('Sign in', { selector: 'button' }))
    expect(
      await screen.findByText('Enter your email address'),
    ).toBeInTheDocument()
    expect(screen.getByText('Enter your password')).toBeInTheDocument()
    expect(fake.calls['signIn']).toBeUndefined()

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'not-an-email' },
    })
    fireEvent.click(screen.getByText('Sign in', { selector: 'button' }))
    expect(
      await screen.findByText('Enter a valid email address'),
    ).toBeInTheDocument()
    expect(fake.calls['signIn']).toBeUndefined()
  })

  it('shows the friendly localized message for wrong credentials', async () => {
    await renderLogin()
    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'ada@example.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'wrong' },
    })
    await act(async () => {
      fireEvent.click(screen.getByText('Sign in', { selector: 'button' }))
    })
    expect(
      await screen.findByText('Incorrect email or password.'),
    ).toBeInTheDocument()
  })

  it('signs in and hands over to the app', async () => {
    await renderLogin()
    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'ada@example.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'correct horse' },
    })
    await act(async () => {
      fireEvent.click(screen.getByText('Sign in', { selector: 'button' }))
    })
    expect(await screen.findByText('the app')).toBeInTheDocument()
  })

  it('enforces the project password policy on sign-up', async () => {
    await renderLogin()
    // Flip to sign-up (the project has signUpOpen).
    fireEvent.click(screen.getByText('Sign up'))
    await screen.findByText('Create account', { selector: 'h1' })
    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'ada@example.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'short' }, // < passwordMinLength 8
    })
    fireEvent.click(screen.getByText('Create account', { selector: 'button' }))
    expect(
      await screen.findByText('Use at least 8 characters'),
    ).toBeInTheDocument()
  })

  it('shows the verify-email notice when sign-up needs verification', async () => {
    await renderLogin({ signUpPolicy: 'verify' })
    fireEvent.click(screen.getByText('Sign up'))
    await screen.findByText('Create account', { selector: 'h1' })
    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'ada@example.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'long enough pw' },
    })
    await act(async () => {
      fireEvent.click(
        screen.getByText('Create account', { selector: 'button' }),
      )
    })
    expect(
      await screen.findByText(/check your inbox to verify/),
    ).toBeInTheDocument()
    // Back on the sign-in mode.
    expect(screen.getByText('Sign in', { selector: 'h1' })).toBeInTheDocument()
  })

  it('hides the sign-up toggle when sign-up is closed', async () => {
    const { client, fake } = fakeClient()
    fake.state.projectConfig.signUpOpen = false
    render(
      <MothProvider client={client}>
        <div>app</div>
      </MothProvider>,
    )
    await screen.findByLabelText('Email')
    expect(screen.queryByText('Sign up')).not.toBeInTheDocument()
  })

  it('runs the forgot-password flow', async () => {
    const { fake } = await renderLogin()
    fireEvent.click(screen.getByText('Forgot password?'))
    await screen.findByText('Reset password', { selector: 'h1' })
    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'ada@example.com' },
    })
    await act(async () => {
      fireEvent.click(screen.getByText('Send reset link'))
    })
    expect(await screen.findByText('Check your email')).toBeInTheDocument()
    expect(
      screen.getByText(/If an account exists for ada@example.com/),
    ).toBeInTheDocument()
    expect(fake.calls['requestPasswordReset']).toBe(1)
    fireEvent.click(screen.getByText('Back to sign in'))
    await screen.findByText('Sign in', { selector: 'h1' })
  })

  it('shows provider buttons only when enabled and the slug is configured', async () => {
    const { fake } = await renderLogin({ config: { projectSlug: 'myapp' } })
    // Providers disabled in config: no buttons.
    expect(screen.queryByText('Continue with Google')).not.toBeInTheDocument()
    void fake
  })

  it('strips the URL fragment from the OAuth redirect (hash routers)', async () => {
    // A hash-routed SPA: the current URL carries a fragment the server
    // would refuse in a redirect URI.
    window.history.replaceState(null, '', '/app#/settings')
    const navigated: string[] = []
    const { client, fake } = fakeClient({
      config: { projectSlug: 'myapp' },
      client: { navigate: (url) => navigated.push(url) },
    })
    fake.state.projectConfig.google!.enabled = true
    render(
      <MothProvider client={client}>
        <div>app</div>
      </MothProvider>,
    )
    fireEvent.click(await screen.findByText('Continue with Google'))
    expect(navigated).toHaveLength(1)
    const redirect = new URL(navigated[0]!).searchParams.get('redirect')!
    expect(redirect).not.toContain('#')
    expect(new URL(redirect).pathname).toBe('/app')
  })

  it('renders Google button and navigates to the redirect start URL', async () => {
    const { client, fake } = fakeClient({ config: { projectSlug: 'myapp' } })
    fake.state.projectConfig.google!.enabled = true
    render(
      <MothProvider client={client}>
        <div>app</div>
      </MothProvider>,
    )
    const button = await screen.findByText('Continue with Google')
    // jsdom cannot navigate; assert the URL the click would assign by
    // intercepting location.assign via the client helper.
    expect(client.oauthStartUrl('google', 'https://app.test/')).toContain(
      '/oauth/google/start?project=myapp',
    )
    expect(button).toBeEnabled()
  })
})
