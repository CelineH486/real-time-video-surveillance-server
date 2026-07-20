import 'package:flutter/material.dart';

import '../models/truck.dart';
import '../services/api_client.dart';

class TruckSelectScreen extends StatefulWidget {
  const TruckSelectScreen({
    super.key,
    required this.apiClient,
    required this.onLogout,
  });

  final ApiClient apiClient;
  final Future<void> Function() onLogout;

  @override
  State<TruckSelectScreen> createState() => _TruckSelectScreenState();
}

class _TruckSelectScreenState extends State<TruckSelectScreen> {
  late Future<List<Truck>> _trucks;

  @override
  void initState() {
    super.initState();
    _trucks = widget.apiClient.getTrucks();
  }

  void _reload() {
    setState(() {
      _trucks = widget.apiClient.getTrucks();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('選擇車機'),
        actions: [
          IconButton(
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
            tooltip: '重新整理',
          ),
          IconButton(
            onPressed: widget.onLogout,
            icon: const Icon(Icons.logout),
            tooltip: '登出',
          ),
        ],
      ),
      body: FutureBuilder<List<Truck>>(
        future: _trucks,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return _ErrorState(message: '${snapshot.error}', onRetry: _reload);
          }

          final trucks = snapshot.data ?? const [];
          if (trucks.isEmpty) {
            return _ErrorState(message: '目前沒有車機資料', onRetry: _reload);
          }

          return ListView.separated(
            padding: const EdgeInsets.all(16),
            itemCount: trucks.length,
            separatorBuilder: (context, index) => const SizedBox(height: 12),
            itemBuilder: (context, index) {
              final truck = trucks[index];
              return Card(
                child: ListTile(
                  contentPadding: const EdgeInsets.all(16),
                  leading: CircleAvatar(
                    backgroundColor: truck.status == 'online'
                        ? const Color(0xFF35E6A5)
                        : Colors.grey.shade700,
                    child: const Icon(Icons.local_shipping),
                  ),
                  title: Text(truck.truckId.toUpperCase()),
                  subtitle: Text(
                    '${truck.plateNo} · ${_statusLabel(truck.status)}',
                  ),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () {
                    Navigator.of(
                      context,
                    ).pushNamed('/trucks/${truck.truckId}/cameras');
                  },
                ),
              );
            },
          );
        },
      ),
    );
  }
}

String _statusLabel(String status) => status == 'online' ? '在線' : '離線';

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.cloud_off, size: 48),
            const SizedBox(height: 12),
            Text(message, textAlign: TextAlign.center),
            const SizedBox(height: 16),
            FilledButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh),
              label: const Text('重新整理'),
            ),
          ],
        ),
      ),
    );
  }
}
