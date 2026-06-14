import { apiClient } from './api';

// Mock fetch globally
global.fetch = jest.fn();

describe('apiClient', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  test('getHealth returns health data', async () => {
    const mockHealth = { status: 'healthy', service: 'test' };
    global.fetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockHealth,
    });

    const result = await apiClient.getHealth();
    expect(result).toEqual(mockHealth);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/health')
    );
  });

  test('getUsers returns users list', async () => {
    const mockUsers = { users: [{ id: 1, username: 'test' }] };
    global.fetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockUsers,
    });

    const result = await apiClient.getUsers();
    expect(result).toEqual(mockUsers);
  });

  test('login returns token on success', async () => {
    const mockResponse = { token: 'test_token', expires_at: '2026-01-01' };
    global.fetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    const result = await apiClient.login('alice', 'password');
    expect(result).toEqual(mockResponse);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/login'),
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ username: 'alice', password: 'password' }),
      })
    );
  });

  test('login throws error on failure', async () => {
    global.fetch.mockResolvedValueOnce({
      ok: false,
      json: async () => ({ error: 'Invalid credentials' }),
    });

    await expect(apiClient.login('wrong', 'wrong')).rejects.toThrow(
      'Invalid credentials'
    );
  });
});
