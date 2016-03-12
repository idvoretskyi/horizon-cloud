package ssh

import "testing"

func TestValidKey(t *testing.T) {
	tests := []struct {
		Key   string
		Valid bool
	}{
		{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDFJYYZRAakqyzg9Fy6nuyxvJe4eNOT4AG8bfZH7EB2HcHLR6OmnhwQsE4fwx878eeFMwuQYkeU/fW3/5VgqLhTHB4Za8C4ZmwN4RvAZbidMf53+5FuwX6bTY6OZcDwsIiD1rss/+M7PcwHE0Ig8/UgBCb38amFAWPUgyELfd/+ZKDlxBRETH3Ia0+UOR/JYf8Xl6XWR+xCGgIY3AI8n6yQsusCaoKMlK2somn6NXBIJ+2DejgdCGeEj1/yu4lM2UMBwuPuoaBOJbjBNhKaQOUIK4P/50mY/cpTCLFLVxgftIc3aZgnai04DIVAe5PmfXRl7i6AbJgYHvEfqmKjCNYz ubuntu@ip-10-0-0-175", false},
		{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDFJYYZRAakqyzg9Fy6nuyxvJe4eNOT4AG8bfZH7EB2HcHLR6OmnhwQsE4fwx878eeFMwuQYkeU/fW3/5VgqLhTHB4Za8C4ZmwN4RvAZbidMf53+5FuwX6bTY6OZcDwsIiD1rss/+M7PcwHE0Ig8/UgBCb38amFAWPUgyELfd/+ZKDlxBRETH3Ia0+UOR/JYf8Xl6XWR+xCGgIY3AI8n6yQsusCaoKMlK2somn6NXBIJ+2DejgdCGeEj1/yu4lM2UMBwuPuoaBOJbjBNhKaQOUIK4P/50mY/cpTCLFLVxgftIc3aZgnai04DIVAe5PmfXRl7i6AbJgYHvEfqmKjCNYz", false},
		{"AAAAB3NzaC1yc2EAAAADAQABAAABAQDFJYYZRAakqyzg9Fy6nuyxvJe4eNOT4AG8bfZH7EB2HcHLR6OmnhwQsE4fwx878eeFMwuQYkeU/fW3/5VgqLhTHB4Za8C4ZmwN4RvAZbidMf53+5FuwX6bTY6OZcDwsIiD1rss/+M7PcwHE0Ig8/UgBCb38amFAWPUgyELfd/+ZKDlxBRETH3Ia0+UOR/JYf8Xl6XWR+xCGgIY3AI8n6yQsusCaoKMlK2somn6NXBIJ+2DejgdCGeEj1/yu4lM2UMBwuPuoaBOJbjBNhKaQOUIK4P/50mY/cpTCLFLVxgftIc3aZgnai04DIVAe5PmfXRl7i6AbJgYHvEfqmKjCNYz comment", false},
		{"AAAAB3NzaC1yc2EAAAADAQABAAABAQDFJYYZRAakqyzg9Fy6nuyxvJe4eNOT4AG8bfZH7EB2HcHLR6OmnhwQsE4fwx878eeFMwuQYkeU/fW3/5VgqLhTHB4Za8C4ZmwN4RvAZbidMf53+5FuwX6bTY6OZcDwsIiD1rss/+M7PcwHE0Ig8/UgBCb38amFAWPUgyELfd/+ZKDlxBRETH3Ia0+UOR/JYf8Xl6XWR+xCGgIY3AI8n6yQsusCaoKMlK2somn6NXBIJ+2DejgdCGeEj1/yu4lM2UMBwuPuoaBOJbjBNhKaQOUIK4P/50mY/cpTCLFLVxgftIc3aZgnai04DIVAe5PmfXRl7i6AbJgYHvEfqmKjCNYz", true},
		{"AAAA", false},
	}

	for _, test := range tests {
		out := ValidKey(test.Key)
		if out != test.Valid {
			t.Errorf("ValidKey(%#v) = %#v, but wanted %#v",
				test.Key, out, test.Valid)
		}
	}
}
