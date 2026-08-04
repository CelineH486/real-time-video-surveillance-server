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
    setState(() => _trucks = widget.apiClient.getTrucks());
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        toolbarHeight: 82,
        title: const _BrandTitle(title: '選擇車機'),
        actions: [
          IconButton(
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
            tooltip: '重新整理',
          ),
          const SizedBox(width: 4),
          Padding(
            padding: const EdgeInsets.only(right: 16),
            child: OutlinedButton.icon(
              onPressed: widget.onLogout,
              icon: const Icon(Icons.logout),
              label: const Text('登出'),
            ),
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
            return _ErrorState(message: '目前沒有可用的車機', onRetry: _reload);
          }

          return LayoutBuilder(
            builder: (context, constraints) {
              final columns = constraints.maxWidth >= 1100
                  ? 3
                  : constraints.maxWidth >= 700
                  ? 2
                  : 1;
              return GridView.builder(
                padding: EdgeInsets.symmetric(
                  horizontal: constraints.maxWidth >= 700 ? 32 : 16,
                  vertical: 24,
                ),
                gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                  crossAxisCount: columns,
                  mainAxisSpacing: 16,
                  crossAxisSpacing: 16,
                  mainAxisExtent: 132,
                ),
                itemCount: trucks.length,
                itemBuilder: (context, index) {
                  final truck = trucks[index];
                  final online = truck.status == 'online';
                  return Card(
                    clipBehavior: Clip.antiAlias,
                    child: InkWell(
                      onTap: () => Navigator.of(
                        context,
                      ).pushNamed('/trucks/${truck.truckId}/cameras'),
                      child: Padding(
                        padding: const EdgeInsets.all(20),
                        child: Row(
                          children: [
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                mainAxisAlignment: MainAxisAlignment.center,
                                children: [
                                  Text(
                                    truck.truckId.toUpperCase(),
                                    style: const TextStyle(
                                      color: Color(0xFF35E6A5),
                                      fontWeight: FontWeight.w800,
                                      letterSpacing: 1.5,
                                    ),
                                  ),
                                  const SizedBox(height: 8),
                                  Text(
                                    truck.plateNo,
                                    style: Theme.of(
                                      context,
                                    ).textTheme.titleLarge,
                                  ),
                                ],
                              ),
                            ),
                            _StatusChip(online: online),
                            const SizedBox(width: 4),
                            const Icon(Icons.chevron_right),
                          ],
                        ),
                      ),
                    ),
                  );
                },
              );
            },
          );
        },
      ),
    );
  }
}

class _BrandTitle extends StatelessWidget {
  const _BrandTitle({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        const Text(
          'REAL-TIME SURVEILLANCE',
          style: TextStyle(
            color: Color(0xFF35E6A5),
            fontSize: 10,
            letterSpacing: 1.8,
            fontWeight: FontWeight.w800,
          ),
        ),
        Text(title),
      ],
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.online});

  final bool online;

  @override
  Widget build(BuildContext context) {
    final color = online ? const Color(0xFF35E6A5) : Colors.redAccent;
    return Chip(
      label: Text(online ? '在線' : '離線'),
      side: BorderSide.none,
      labelStyle: TextStyle(color: color, fontWeight: FontWeight.w700),
      backgroundColor: color.withValues(alpha: .12),
    );
  }
}

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
