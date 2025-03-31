export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export const handleApiError = (error: unknown): string => {
  if (error instanceof ApiError) {
    return error.message;
  }

  if (error instanceof Error) {
    // Handle validation errors
    if (error.message.includes('validation failed')) {
      return 'Please check your input and try again.';
    }
    // Handle network errors
    if (error.message.includes('Failed to fetch')) {
      return 'Unable to connect to the server. Please check your connection.';
    }
    return error.message;
  }

  return 'An unexpected error occurred. Please try again.';
};

export const getErrorMessage = (error: unknown): string => {
  if (error instanceof ApiError) {
    switch (error.status) {
      case 400:
        return 'Invalid request. Please check your input.';
      case 401:
        return 'Unauthorized. Please log in again.';
      case 403:
        return 'You do not have permission to perform this action.';
      case 404:
        return 'The requested resource was not found.';
      case 409:
        return 'A conflict occurred. Please try again.';
      case 422:
        return 'Validation failed. Please check your input.';
      case 429:
        return 'Too many requests. Please try again later.';
      case 500:
        return 'Server error. Please try again later.';
      default:
        return error.message;
    }
  }
  return handleApiError(error);
}; 